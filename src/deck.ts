import { PowerPlant, Resource } from "./types";

export const powerPlants: PowerPlant[] = [
    {
      initialCost: 3,
      resourcesRequired: 2,
      resourceType: Resource.OIL,
      housesPowered: 1
    },
    {
      initialCost: 4,
      resourcesRequired: 2,
      resourceType: Resource.COAL,
      housesPowered: 1
    },
    {
      initialCost: 5,
      resourcesRequired: 2,
      resourceType: Resource.HYBRID,
      housesPowered: 1
    },
    {
      initialCost: 6,
      resourcesRequired: 1,
      resourceType: Resource.TRASH,
      housesPowered: 1
    },
    {
      initialCost: 7,
      resourcesRequired: 3,
      resourceType: Resource.OIL,
      housesPowered: 2
    },
    {
      initialCost: 8,
      resourcesRequired: 3,
      resourceType: Resource.COAL,
      housesPowered: 2
    },
    {
      initialCost: 9,
      resourcesRequired: 1,
      resourceType: Resource.OIL,
      housesPowered: 1
    },
    {
      initialCost: 10,
      resourcesRequired: 2,
      resourceType: Resource.COAL,
      housesPowered: 2
    },
    {
      initialCost: 11,
      resourcesRequired: 1,
      resourceType: Resource.URANIUM,
      housesPowered: 2
    },
    {
      initialCost: 12,
      resourcesRequired: 2,
      resourceType: Resource.HYBRID,
      housesPowered: 2
    },
    {
      initialCost: 13,
      resourcesRequired: 0,
      resourceType: Resource.GREEN,
      housesPowered: 1
    },
    {
      initialCost: 14,
      resourcesRequired: 2,
      resourceType: Resource.TRASH,
      housesPowered: 2
    },
    {
      initialCost: 15,
      resourcesRequired: 2,
      resourceType: Resource.COAL,
      housesPowered: 3
    },
    {
      initialCost: 16,
      resourcesRequired: 2,
      resourceType: Resource.OIL,
      housesPowered: 3
    },
    {
      initialCost: 17,
      resourcesRequired: 1,
      resourceType: Resource.URANIUM,
      housesPowered: 2
    },
    {
      initialCost: 18,
      resourcesRequired: 0,
      resourceType: Resource.GREEN,
      housesPowered: 2
    },
    {
      initialCost: 19,
      resourcesRequired: 2,
      resourceType: Resource.TRASH,
      housesPowered: 3
    },
    {
      initialCost: 20,
      resourcesRequired: 3,
      resourceType: Resource.COAL,
      housesPowered: 5
    },
    {
      initialCost: 21,
      resourcesRequired: 2,
      resourceType: Resource.HYBRID,
      housesPowered: 4
    },
    {
      initialCost: 22,
      resourcesRequired: 0,
      resourceType: Resource.GREEN,
      housesPowered: 2
    },
    {
      initialCost: 23,
      resourcesRequired: 1,
      resourceType: Resource.URANIUM,
      housesPowered: 3
    },
    {
      initialCost: 24,
      resourcesRequired: 2,
      resourceType: Resource.TRASH,
      housesPowered: 4
    },
    {
      initialCost: 25,
      resourcesRequired: 2,
      resourceType: Resource.COAL,
      housesPowered: 5
    },
    {
      initialCost: 26,
      resourcesRequired: 2,
      resourceType: Resource.OIL,
      housesPowered: 5
    },
    {
      initialCost: 27,
      resourcesRequired: 0,
      resourceType: Resource.GREEN,
      housesPowered: 3
    },
    {
      initialCost: 28,
      resourcesRequired: 1,
      resourceType: Resource.URANIUM,
      housesPowered: 4
    },
    {
      initialCost: 29,
      resourcesRequired: 1,
      resourceType: Resource.HYBRID,
      housesPowered: 4
    },
    {
      initialCost: 30,
      resourcesRequired: 3,
      resourceType: Resource.TRASH,
      housesPowered: 6
    },
    {
      initialCost: 31,
      resourcesRequired: 3,
      resourceType: Resource.COAL,
      housesPowered: 6
    },
    {
      initialCost: 32,
      resourcesRequired: 3,
      resourceType: Resource.OIL,
      housesPowered: 6
    },
    {
      initialCost: 33,
      resourcesRequired: 0,
      resourceType: Resource.GREEN,
      housesPowered: 4
    },
    {
      initialCost: 34,
      resourcesRequired: 1,
      resourceType: Resource.URANIUM,
      housesPowered: 5
    },
    {
      initialCost: 35,
      resourcesRequired: 1,
      resourceType: Resource.OIL,
      housesPowered: 5
    },
    {
      initialCost: 36,
      resourcesRequired: 3,
      resourceType: Resource.COAL,
      housesPowered: 7
    },
    {
      initialCost: 37,
      resourcesRequired: 0,
      resourceType: Resource.GREEN,
      housesPowered: 4
    },
    {
      initialCost: 38,
      resourcesRequired: 3,
      resourceType: Resource.TRASH,
      housesPowered: 7
    },
    {
      initialCost: 39,
      resourcesRequired: 1,
      resourceType: Resource.URANIUM,
      housesPowered: 6
    },
    {
      initialCost: 40,
      resourcesRequired: 2,
      resourceType: Resource.OIL,
      housesPowered: 6
    },
    {
      initialCost: 42,
      resourcesRequired: 2,
      resourceType: Resource.COAL,
      housesPowered: 6
    },
    {
      initialCost: 44,
      resourcesRequired: 0,
      resourceType: Resource.GREEN,
      housesPowered: 5
    },
    {
      initialCost: 46,
      resourcesRequired: 3,
      resourceType: Resource.HYBRID,
      housesPowered: 7
    },
    {
      initialCost: 50,
      resourcesRequired: 0,
      resourceType: Resource.GREEN,
      housesPowered: 6
    },
];

export const shuffleDeck = (inCards: PowerPlant[]): PowerPlant[] => {
  const cards = inCards.slice(0);
  for (let i = cards.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [cards[i], cards[j]] = [cards[j], cards[i]];
  }
  return cards;
}

export const getPowerPlant = (cost: number): PowerPlant => {
  const pp = powerPlants.find(powerPlant => powerPlant.initialCost == cost)
  if (pp) {
    return pp;
  } else { 
    throw new Error("cant find powerPlant");
  };
}