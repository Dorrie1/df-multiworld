# df-multiworld
A simple implementation of world transfer in [dragonfly-mc](https://github.com/df-mc/dragonfly).

This package provides:
- A `WorldProvider` to manage multiple worlds.
- Helper functions to transfer entities (especially players) across worlds.
- A built-in `/multiworld` (alias `/mw`) command with subcommands for teleporting.

---

## Installation

```go
import (
  "fmt"
  "log/slog"

  // Automatically register the command and load all worlds
  // just by importing the package
  "github.com/redstonecraftgg/df-multiworld"

  "github.com/df-mc/dragonfly/server"
)

func main() {
  conf, err := server.DefaultConfig().Config(slog.Default())
  if err != nil {
    panic(err)
  }
  srv := conf.New()
  srv.CloseOnProgramEnd()

  // Manually register the original world created by the server
  // using the folder name inside the root path
  multiworld.Worlds.Worlds["world"] = srv.World()

  srv.Listen()
  for p := range srv.Accept() {
    _ = p
  }
}
```

## Usage

### With function

If you want to move a player programmatically:

```go
multiworld.TransferPlayer(playerObj, Worlds, "world2", mgl64.Vec3{0, 64, 0})
```

This will:

- Save the player's entity handle.

- Remove it from the current world.

- Re-add it to the target world.

- Teleport it to the given position.

---

### With commands

The package also registers a command automatically:

- `/multiworld teleport <world>`

- Alias: `/mw tp <world>`

This will transfer the player to the given world.

**Important**: make sure the target position is not obstructed (e.g. no solid block), otherwise the player may suffocate.

The default teleport position is: 0, 64, 0

## Features

- Simple `MapWorldProvider` using Go `map[string]*world.World`.

- TransferPlayer helper with proper entity handling.

- Autocomplete support for world names in chat.

## Notes

- Only loaded worlds can be teleported to.

- If you want custom spawn locations, you can extend `TransferPlayer` with your own logic.

- Currently designed for player transfers, but can be adapted for other entities if possible.
