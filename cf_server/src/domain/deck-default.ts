/**
 * Default power plant deck. Values match go_server/internal/game/powerplant.go so
 * existing clients see the same plants. Seeded into D1 (`power_plants`, deck "standard")
 * by scripts/build-seed.ts; the engine reads the deck from D1 and falls back to this.
 */
export type PlantResourceType = "Coal" | "Oil" | "Garbage" | "Uranium" | "Hybrid" | "Wind"

export interface PlantDef {
  readonly number: number
  readonly capacity: number
  readonly resourceType: PlantResourceType
  readonly resourceCost: number
}

const p = (number: number, capacity: number, resourceType: PlantResourceType, resourceCost: number): PlantDef => ({
  number,
  capacity,
  resourceType,
  resourceCost
})

export const DEFAULT_DECK: ReadonlyArray<PlantDef> = [
  p(3, 1, "Oil", 2), p(4, 1, "Coal", 2), p(5, 1, "Hybrid", 2), p(6, 1, "Garbage", 1),
  p(7, 2, "Oil", 3), p(8, 2, "Coal", 3), p(9, 1, "Oil", 1), p(10, 2, "Coal", 2),
  p(11, 2, "Uranium", 1), p(12, 2, "Hybrid", 2), p(13, 1, "Wind", 0), p(14, 2, "Garbage", 2),
  p(15, 3, "Coal", 2), p(16, 3, "Oil", 2), p(17, 2, "Uranium", 1), p(18, 2, "Wind", 0),
  p(19, 3, "Garbage", 2), p(20, 5, "Coal", 3), p(21, 4, "Hybrid", 2), p(22, 2, "Wind", 0),
  p(23, 3, "Uranium", 1), p(24, 4, "Garbage", 2), p(25, 5, "Coal", 2), p(26, 5, "Oil", 2),
  p(27, 3, "Wind", 0), p(28, 4, "Uranium", 1), p(29, 4, "Hybrid", 1), p(30, 6, "Garbage", 3),
  p(31, 6, "Coal", 3), p(32, 6, "Oil", 3), p(33, 4, "Wind", 0), p(34, 5, "Uranium", 1),
  p(35, 5, "Oil", 1), p(36, 7, "Coal", 3), p(37, 4, "Wind", 0), p(38, 7, "Garbage", 3),
  p(39, 6, "Uranium", 1), p(40, 6, "Oil", 2), p(42, 6, "Coal", 2), p(44, 5, "Wind", 0),
  p(46, 7, "Hybrid", 3), p(50, 6, "Wind", 0)
]

export const DEFAULT_ACHIEVEMENTS: ReadonlyArray<{
  id: string
  name: string
  description: string
  category: string
  icon: string
  points: number
  criteria: string
}> = [
  { id: "first_win", name: "First Victory", description: "Win your first game", category: "victory", icon: "🏆", points: 10, criteria: "games_won >= 1" },
  { id: "hat_trick", name: "Hat Trick", description: "Win 3 games in a row", category: "victory", icon: "🎩", points: 25, criteria: "winning_streak >= 3" },
  { id: "unstoppable", name: "Unstoppable", description: "Win 5 games in a row", category: "victory", icon: "🔥", points: 50, criteria: "winning_streak >= 5" },
  { id: "master_strategist", name: "Master Strategist", description: "Win 50 games", category: "victory", icon: "🧠", points: 100, criteria: "games_won >= 50" },
  { id: "money_bags", name: "Money Bags", description: "End a game with 200+ elektro", category: "economic", icon: "💰", points: 20, criteria: "final_money >= 200" },
  { id: "resource_hoarder", name: "Resource Hoarder", description: "Own 20+ resources at once", category: "economic", icon: "📦", points: 15, criteria: "max_resources >= 20" },
  { id: "eco_warrior", name: "Eco Warrior", description: "Win using only renewable power plants", category: "economic", icon: "🌱", points: 30, criteria: "renewable_only_win" },
  { id: "city_builder", name: "City Builder", description: "Build in 15+ cities in a single game", category: "expansion", icon: "🏙️", points: 25, criteria: "max_cities >= 15" },
  { id: "rapid_expansion", name: "Rapid Expansion", description: "Build in 10 cities within 5 rounds", category: "expansion", icon: "⚡", points: 30, criteria: "rapid_expansion" },
  { id: "monopolist", name: "Monopolist", description: "Control all cities in a region", category: "expansion", icon: "👑", points: 35, criteria: "region_monopoly" },
  { id: "plant_collector", name: "Plant Collector", description: "Own 5 power plants at once", category: "plants", icon: "🏭", points: 20, criteria: "max_plants >= 5" },
  { id: "high_capacity", name: "High Capacity", description: "Own a plant that powers 7+ cities", category: "plants", icon: "⚡", points: 15, criteria: "max_plant_capacity >= 7" },
  { id: "diversified", name: "Diversified Portfolio", description: "Own plants of 4 different resource types", category: "plants", icon: "🎨", points: 25, criteria: "resource_diversity >= 4" },
  { id: "underdog", name: "Underdog Victory", description: "Win from last place in turn order", category: "special", icon: "🐕", points: 40, criteria: "underdog_win" },
  { id: "perfectionist", name: "Perfectionist", description: "Win by powering all your cities", category: "special", icon: "✨", points: 30, criteria: "perfect_power" },
  { id: "speed_demon", name: "Speed Demon", description: "Win a game in under 30 minutes", category: "special", icon: "🏎️", points: 35, criteria: "speed_win" },
  { id: "regular_player", name: "Regular Player", description: "Play 25 games", category: "participation", icon: "🎮", points: 15, criteria: "games_played >= 25" },
  { id: "dedicated_player", name: "Dedicated Player", description: "Play 100 games", category: "participation", icon: "🌟", points: 50, criteria: "games_played >= 100" },
  { id: "map_explorer", name: "Map Explorer", description: "Play on all available maps", category: "participation", icon: "🗺️", points: 25, criteria: "all_maps_played" }
]
