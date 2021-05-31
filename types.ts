
enum Resource {
  COAL = "COAL",
  OIL = "OIL",
  HYBRID = "HYBRID",
  TRASH = "TRASH",
  URANIUM = "URANIUM",
  GREEN = "GREEN",
};

interface PowerPlant {
  initialCost: number;
  resourcesRequired: number;
  resourseType: Resource;
  housesPowered: number;
}

interface ResourceState {
  resourceType: Resource;
  available: {
    cost: number;
    quantity: number;
  }[]
}

interface GameState {
  id: string;
  players: PlayerState[];
  market: PowerPlant[];
  deck: PowerPlant[];
  discard: PowerPlant[];
  gamePhase: number;
  roundStep: number;
  cities: CityState[];
  resources: ResourceState[];
}

interface BidState {
  currentBidder: string;
  remainingBidders: string[];
  eligibleBidders: string[];
  currentBid: number;
  powerPlant: PowerPlant;
}

interface PowerPlantState {
  powerPlant: PowerPlant;
  currentResources: {
    resourceType: Resource;
    quantity: number;
  }[];
}

interface PlayerState {
  id: string;
  money: number;
  powerPlants: PowerPlant[];
  cities: string[];
}

interface CityState {
  name: string;
  region: number;
  location: {
    x: number;
    y: number;
  };
  connections: {
    name: string;
    cost: number;
    location: {
      x: number;
      y: number;
    };
  }[];
  houses: string[];
}