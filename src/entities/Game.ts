import { Collection, Entity, OneToMany, Property } from "@mikro-orm/core";
import { Player } from ".";
import { BaseEntity } from "./BaseEntity";

@Entity()
export class Game extends BaseEntity {

    @Property()
    code: string;

    @Property()
    turnOrder!: string[];

    @Property()
    gamePhase: number;

    @Property()
    roundStep: number;

    @OneToMany(() => Player, p => p.game)
    players = new Collection<Player>(this);

    constructor(code: string) {
        super();
        this.code = code;
        this.gamePhase = 0;
        this.roundStep = 0;
    }

}