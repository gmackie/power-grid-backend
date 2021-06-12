import { Player } from '../entities';
import { PowerPlant } from '../types';

export const startBid = (player: string, powerPlant: PowerPlant, value: number = powerPlant.initialCost) => {
  // see if it is players turn
  // see if powerPlant is valid in market
  // make sure player can pay cost
  // determine bidders
  // set bid state
};

export const makeBid = (player: string, value: number) => {
  // see if it is players turn to bid
  // make sure player can pay cost
  // see if value is > currentBid
  // if higher, update BidState
  // if 0 (pass), remove from eligibleBidders
  // if highest bidder is only one left, give powerPlant
};

const grantPowerPlant = (player: Player, powerPlant: PowerPlant) => {
  // remove powerPlant from market
  // decrement player cash
  // give powerPlant to player
  // add new powerPlant to market
};