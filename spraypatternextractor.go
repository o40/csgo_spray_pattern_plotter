package main

import (
	"fmt"
	"os"
	"flag"
	"sort"

	dem "github.com/markus-wa/demoinfocs-golang"
	events "github.com/markus-wa/demoinfocs-golang/events"
)

// Settings
const numPaddingFrames = 15

type WeaponFiredData struct {
	IngameTick int
	WeaponId int
}

type ViewDirectionData struct {
	IngameTick int
	WeaponId int
	ViewDirectionX float32
	ViewDirectionY float32
}

type SprayData struct {
	WeaponFired bool
	WeaponHit bool
	Kill bool
	WeaponId int
	ViewDirectionX float32
	ViewDirectionY float32
}

func outputSprayPatternAsCsv(parser *dem.Parser) {

	// Store view direction, weapon id and tick for all ticks where a weapon was fired (per player)
	weaponFiredDataPerPlayer := make(map[string][]WeaponFiredData)
	parser.RegisterEventHandler(func(e events.WeaponFire) {
		// Only check rifles
		if (e.Weapon.Weapon >= 300 && e.Weapon.Weapon < 400) {
			var weaponFiredData WeaponFiredData
			weaponFiredData.IngameTick = parser.GameState().IngameTick()
			weaponFiredData.WeaponId = int(e.Weapon.Weapon)
			weaponFiredDataPerPlayer[e.Shooter.Name] = append(weaponFiredDataPerPlayer[e.Shooter.Name], weaponFiredData)
		}
	})

	// Get ticks where a player hurts another player
	weaponHitDataPerPlayer := make(map[string][]int)
	parser.RegisterEventHandler(func(e events.PlayerHurt) {
		if e.Attacker != nil && e.Player != nil {
			weaponHitDataPerPlayer[e.Attacker.Name] = append(weaponHitDataPerPlayer[e.Attacker.Name], parser.GameState().IngameTick())
		}
	})

	// Get ticks where a player kills another player
	killTicksPerPlayer := make(map[string][]int)
	parser.RegisterEventHandler(func(e events.Kill) {
		if e.Killer != nil {
			killTicksPerPlayer[e.Killer.Name] = append(killTicksPerPlayer[e.Killer.Name], parser.GameState().IngameTick())
		}
	})

	// Get player view angles per tick and store per player
	viewDirectionDataPerPlayer := make(map[string][]ViewDirectionData)
	parser.RegisterEventHandler(func(events.TickDone) {
		players := parser.GameState().Participants().Playing()
		for _, player := range players {
			if player != nil {
				var viewDirectionData ViewDirectionData
				viewDirectionData.IngameTick =  parser.GameState().IngameTick()
				viewDirectionData.ViewDirectionX = player.ViewDirectionX
				viewDirectionData.ViewDirectionY = player.ViewDirectionY
				viewDirectionData.WeaponId = player.ActiveWeaponID
				viewDirectionDataPerPlayer[player.Name] = append(viewDirectionDataPerPlayer[player.Name], viewDirectionData)
			}
		}
	})

	// Run parser (to populate weaponFiredData and viewDirectionData)
	parser.ParseToEnd()

	// Dump csv data per player
	for playerKey, weaponFiredSlice := range weaponFiredDataPerPlayer {

		// Save spray data per tick in this inner loop
		sprayDataPerTick := make(map[int]SprayData)

		// Loop over ticks with fired weapons and add those position + padding ticks
		// to create a list of view directions per tick. Init weapon fired to false.
		for _, weaponFiredData := range weaponFiredSlice {
			for _, viewDirectionData := range viewDirectionDataPerPlayer[playerKey] {
				if (viewDirectionData.IngameTick >= weaponFiredData.IngameTick &&
					viewDirectionData.IngameTick <= weaponFiredData.IngameTick + numPaddingFrames) {
					var sprayData SprayData
					sprayData.WeaponFired = false
					sprayData.WeaponHit = false
					sprayData.Kill = false
					sprayData.ViewDirectionX = viewDirectionData.ViewDirectionX
					sprayData.ViewDirectionY = viewDirectionData.ViewDirectionY
					sprayData.WeaponId = weaponFiredData.WeaponId
					sprayDataPerTick[viewDirectionData.IngameTick] = sprayData
				}
			}
		}

		// Set WeaponFired to true in the spray data for the ticks where a weapon was fired
		for _, weaponFiredData := range weaponFiredSlice {
			var sprayData = sprayDataPerTick[weaponFiredData.IngameTick]
            sprayData.WeaponFired = true
            sprayDataPerTick[weaponFiredData.IngameTick] = sprayData
		}

		// Set WeaponHit to true for all ticks where a player hit the shot
		for _, weaponHitTick := range weaponHitDataPerPlayer[playerKey] {
			var sprayData = sprayDataPerTick[weaponHitTick]
			sprayData.WeaponHit = true
			sprayDataPerTick[weaponHitTick] = sprayData
		}

		// Set WeaponHit to true for all ticks where a player hit the shot
		for _, killTick := range killTicksPerPlayer[playerKey] {
			var sprayData = sprayDataPerTick[killTick]
			sprayData.Kill = true
			sprayDataPerTick[killTick] = sprayData
		}

		// Get the tick "keys" and sort them
		var keys []int
    	for k := range sprayDataPerTick {
        	keys = append(keys, k)
    	}
		sort.Ints(keys)

		// Print the spray data per tick in order
		for _, tick := range keys {
			var sprayData = sprayDataPerTick[tick]
			weaponFired := 0
			weaponHit := 0
			kill := 0

			if (sprayData.WeaponFired) {
				weaponFired = 1
			}

			if (sprayData.WeaponHit) {
				weaponHit = 1
			}

			if (sprayData.Kill) {
				kill = 1
			}

			if (sprayData.Kill || sprayData.WeaponHit) && !sprayData.WeaponFired {
				continue
			}

			fmt.Printf("%s,%d,%d,%d,%d,%f,%f,%d\n",
						playerKey,
						tick,
						weaponFired,
						weaponHit,
						kill,
						sprayData.ViewDirectionX,
						sprayData.ViewDirectionY,
						sprayData.WeaponId)
		}
	}
}

func main() {
	demoPathPtr := flag.String("demo", "", "Path to the demo")
	flag.Parse()

	// Mandatory argument
	if *demoPathPtr == "" {
		fmt.Printf("Missing argument demo\n")
		os.Exit(1)
	}

	if _, err := os.Stat(*demoPathPtr); os.IsNotExist(err) {
  		fmt.Printf("%s does not exist\n", *demoPathPtr)
  		os.Exit(1)
	}

	f, err := os.Open(*demoPathPtr)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	p := dem.NewParser(f)

	outputSprayPatternAsCsv(p)
}