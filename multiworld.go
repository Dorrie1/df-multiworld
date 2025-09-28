package multiworld

import (
	"fmt"
	"os"
	"strings"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/mcdb"
	"github.com/go-gl/mathgl/mgl64"
)

// =====================
// Command Section
// =====================

// MultiWorldList is a command structure for listing all loaded worlds.
// Usage example: /world list
type MultiWorldList struct {
	List cmd.SubCommand `cmd:"list"`
}

// MultiWorldTp is a command structure for teleporting to another world.
// Usage example: /world teleport <world_name>
type MultiWorldTp struct {
	Teleport  cmd.SubCommand          `cmd:"teleport"`
	WorldName cmd.Optional[WorldEnum] `cmd:"world,enum"`
}

// MWTp is an alias command structure for teleporting (short form).
// Usage example: /world tp <world_name>
type MWTp struct {
	Tp        cmd.SubCommand          `cmd:"tp"`
	WorldName cmd.Optional[WorldEnum] `cmd:"world,enum"`
}

// WorldEnum is a string type used for world name enumeration in commands.
type WorldEnum string

func (WorldEnum) Type() string {
	return "world_list"
}

func (WorldEnum) Options(src cmd.Source) []string {
	return GetWorldNames()
}

// =====================
// Utility & Function Section
// =====================

// WorldProvider is an interface for world providers, allowing retrieval of worlds by name.
type WorldProvider interface {
	GetWorld(name string) (*world.World, bool)
}

// MapWorldProvider is a simple implementation of WorldProvider using a map.
type MapWorldProvider struct {
	Worlds map[string]*world.World
}

// GetWorld retrieves a world by name from the map.
// Returns the world and true if found, otherwise nil and false.
func (m *MapWorldProvider) GetWorld(name string) (*world.World, bool) {
	w, ok := m.Worlds[name]
	return w, ok
}

// Worlds is a global alias for the MapWorldProvider instance.
var Worlds = &MapWorldProvider{
	Worlds: make(map[string]*world.World),
}

// TransferEntity moves an entity (player or other) from its current world to another world at a given position.
// Example usage: TransferEntity(entityHandle, Worlds, "world2", mgl64.Vec3{0, 64, 0})
func TransferEntity(eH *world.EntityHandle, provider WorldProvider, worldName string, pos mgl64.Vec3) error {
	newWorld, ok := provider.GetWorld(worldName)
	if !ok {
		return fmt.Errorf("world not found: %s", worldName)
	}

	okExec := eH.ExecWorld(func(tx *world.Tx, e world.Entity) {
		tx.RemoveEntity(e)
	})
	if !okExec {
		return fmt.Errorf("failed to execute transaction on world entity (entity closed?)")
	}

	done := make(chan error, 1)
	newWorld.Exec(func(tx *world.Tx) {
		newEnt := tx.AddEntity(eH)
		if newEnt == nil {
			done <- fmt.Errorf("tx.AddEntity returned nil")
			return
		}
		if p, ok := newEnt.(*player.Player); ok {
			p.Teleport(pos)
		}
		done <- nil
	})

	if err := <-done; err != nil {
		return err
	}
	return nil
}

// TransferPlayer is a helper function to move a player to another world at a given position.
// Example usage: TransferPlayer(playerObj, Worlds, "world2", mgl64.Vec3{0, 64, 0})
func TransferPlayer(p *player.Player, provider WorldProvider, worldName string, pos mgl64.Vec3) error {
	return TransferEntity(p.H(), provider, worldName, pos)
}

// LoadWorlds reads and loads all worlds from the "worlds" folder into memory.
// It creates the folder if it does not exist.
// Example usage: LoadWorlds()
func LoadWorlds() {
	if _, err := os.Stat("worlds"); os.IsNotExist(err) {
		if err := os.Mkdir("worlds", 0755); err != nil {
			panic(fmt.Errorf("failed to create worlds folder: %v", err))
		}
	}
	files, err := os.ReadDir("worlds")
	if err != nil {
		panic(fmt.Errorf("failed to read worlds folder: %v", err))
	}

	for _, file := range files {
		if file.IsDir() {
			worldName := file.Name()
			if _, exists := Worlds.Worlds[worldName]; exists {
				continue
			}
			prov, err := mcdb.Open("worlds/" + worldName)
			if err != nil {
				fmt.Printf("failed to open world %s: %v\n", worldName, err)
				continue
			}
			conf := world.Config{Provider: prov}
			w := conf.New()
			Worlds.Worlds[worldName] = w
			fmt.Printf("World %s loaded successfully\n", worldName)
		}
	}
}

// GetWorldNames returns a slice of all loaded world names.
// Example usage: names := GetWorldNames()
func GetWorldNames() []string {
	worlds := make([]string, 0, len(Worlds.Worlds))
	for name := range Worlds.Worlds {
		worlds = append(worlds, name)
	}
	return worlds
}

// =====================
// Command Implementation Section
// =====================

// Run executes the MultiWorldList command, sending the list of available worlds to the player.
// Example: /world list
func (MultiWorldList) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if p, ok := src.(*player.Player); ok {
		names := GetWorldNames()
		p.Messagef("§aWorlds available: §e%s", strings.Join(names, ", "))
	}
}

// Run executes the MultiWorldTp command, teleporting the player to the specified world.
// Example: /world teleport world2
func (c MultiWorldTp) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	p, ok := src.(*player.Player)
	if !ok {
		return
	}

	if worldName, ok := c.WorldName.Load(); ok {
		worldNameStr := string(worldName)
		if worldNameStr == "" {
			p.Messagef("§cUsage: /world teleport <name>")
			return
		}
		eh := p.H()
		p.Messagef("§eSending you to world %s...", worldNameStr)

		go func(eh *world.EntityHandle, wn string) {
			if err := TransferEntity(eh, Worlds, wn, mgl64.Vec3{0, 64, 0}); err != nil {
				_ = eh.ExecWorld(func(tx *world.Tx, e world.Entity) {
					if pp, ok := e.(*player.Player); ok {
						pp.Messagef("§cFailed to move to %s: %v", wn, err)
					}
				})
				return
			}
			_ = eh.ExecWorld(func(tx *world.Tx, e world.Entity) {
				if pp, ok := e.(*player.Player); ok {
					pp.Messagef("§aSuccessfully moved to world §e%s", wn)
				}
			})
		}(eh, worldNameStr)
		return
	}
	p.Messagef("§cUsage: /world teleport <name>")
}

// Run executes the MWTp command, which is an alias for MultiWorldTp.
// Example: /world tp world2
func (c MWTp) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	tpCmd := MultiWorldTp{
		Teleport:  c.Tp,
		WorldName: c.WorldName,
	}
	tpCmd.Run(src, o, tx)
}

// =====================
// Initialization Section
// =====================

// init loads all worlds and registers the multiworld commands.
// This function is called automatically when the package is imported.
func init() {
	LoadWorlds()

	cmd.Register(cmd.New("multiworld", "Multiworld utilities", []string{"mw", "world"}, MultiWorldList{}, MultiWorldTp{}, MWTp{}))
}
