import { Collection, Entity, Index, OneToMany, Property } from "@mikro-orm/core";
import { Player } from ".";
import { powerPlants, shuffleDeck } from "../deck";
import { PowerPlant, BidState, ResourceState, Resource } from "../types";
import { BaseEntity } from "./BaseEntity";

@Entity()
export class Game extends BaseEntity {

  @Index({ name: 'code_index' })
  @Property()
  code: string;

  @Property()
  host: string;

  @Property()
  turnOrder!: string[];

  @Property()
  gamePhase: number;

  @Property()
  roundStep: number;

  @Property()
  deck: PowerPlant[];

  @Property()
  market: PowerPlant[];

  @Property()
  bidState!: BidState;

  @Property()
  resourceState: ResourceState[];

  @OneToMany(() => Player, p => p.game)
  players = new Collection<Player>(this);

  constructor(code: string, host: string) {
    super();
    this.code = code;
    this.host = host;
    this.players.add(new Player(host, this));
    this.gamePhase = 0;
    this.roundStep = 0;
    this.deck = shuffleDeck(powerPlants.slice(8));
    this.market = powerPlants.slice(0,8);
    this.resourceState = [
      {
        resourceType: Resource.COAL,
        available: [
          {
            cost: 1,
            quantity: 3,
          },
          {
            cost: 2,
            quantity: 3,
          },
          {
            cost: 3,
            quantity: 3,
          },
          {
            cost: 4,
            quantity: 3,
          },
          {
            cost: 5,
            quantity: 3,
          },
          {
            cost: 6,
            quantity: 3,
          },
          {
            cost: 7,
            quantity: 3,
          },
          {
            cost: 8,
            quantity: 3,
          },
        ]

      },
      {
        resourceType: Resource.OIL,
        available: [
          {
            cost: 3,
            quantity: 3,
          },
          {
            cost: 4,
            quantity: 3,
          },
          {
            cost: 5,
            quantity: 3,
          },
          {
            cost: 6,
            quantity: 3,
          },
          {
            cost: 7,
            quantity: 3,
          },
          {
            cost: 8,
            quantity: 3,
          },
        ]

      },
      {
        resourceType: Resource.TRASH,
        available: [
          {
            cost: 7,
            quantity: 3,
          },
          {
            cost: 8,
            quantity: 3,
          },
        ]

      },
      {
        resourceType: Resource.URANIUM,
        available: [
          {
            cost: 14,
            quantity: 1,
          },
          {
            cost: 16,
            quantity: 1,
          },
        ]

      },
    ]
    }

}