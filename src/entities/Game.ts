import { Collection, Entity, Index, OneToMany, Property } from "@mikro-orm/core";
import { Player } from ".";
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

    @OneToMany(() => Player, p => p.game)
    players = new Collection<Player>(this);

    constructor(code: string, host: string) {
        super();
        this.code = code;
        this.host = host;
        this.players.add(new Player(host, this));
        this.gamePhase = 0;
        this.roundStep = 0;
    }

}