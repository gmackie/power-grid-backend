import { Entity, ManyToOne, PrimaryKey, Property } from "@mikro-orm/core";
import { Game } from ".";
import { BaseEntity } from "./BaseEntity";

@Entity()
export class Player extends BaseEntity{

    @Property()
    name: string;

    @ManyToOne()
    game: Game;

    @Property()
    money: number;

    @Property()
    houses: string[];

    @Property()
    powerPlants: string[];

    constructor(name: string, game: Game) {
        super();
        this.name = name;
        this.game = game;
        this.money = 50;
        this.houses = [];
        this.powerPlants = [];
    }
}