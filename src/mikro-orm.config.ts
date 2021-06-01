import { Options } from '@mikro-orm/core';
import { Game, Player, BaseEntity } from './entities';

const options: Options = {
    type: 'mongo',
    entities: [Game, Player, BaseEntity],
    dbName: 'power-grid-game',
    debug: true,
};

export default options;