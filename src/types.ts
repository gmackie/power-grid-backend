
export enum Resource {
  COAL = "COAL",
  OIL = "OIL",
  HYBRID = "HYBRID",
  TRASH = "TRASH",
  URANIUM = "URANIUM",
  GREEN = "GREEN",
};

export interface PowerPlant {
  initialCost: number;
  resourcesRequired: number;
  resourceType: Resource;
  housesPowered: number;
}

export interface ResourceState {
  resourceType: Resource;
  available: {
    cost: number;
    quantity: number;
  }[]
}

export interface GameState {
  id: string;
  players: PlayerState[];
  market: PowerPlant[];
  deck: PowerPlant[];
  discard: PowerPlant[];
  gamePhase: number;
  roundStep: number;
  cities: CityState[];
  resources: ResourceState[];
  bidState: BidState;
}

export interface BidState {
  currentBidder: string;
  remainingBidders: string[];
  eligibleBidders: string[];
  currentBid: number;
  powerPlant: PowerPlant;
}

export interface PowerPlantState {
  powerPlant: PowerPlant;
  currentResources: {
    resourceType: Resource;
    quantity: number;
  }[];
}

export interface PlayerState {
  id: string;
  money: number;
  powerPlants: PowerPlantState[];
  cities: string[];
}

export interface CityState {
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
  networks: string[];
  houses: string[];
}