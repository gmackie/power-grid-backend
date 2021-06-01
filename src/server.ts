import express from 'express';
import * as winston from 'winston';
import * as expressWinston from 'express-winston';
import cors from 'cors'
import debug from 'debug';
import { MikroORM, RequestContext, EntityManager, EntityRepository } from '@mikro-orm/core';
import { MongoMemoryServer } from 'mongodb-memory-server';
import { Game, Player } from './entities';
import { GameController, PlayerController } from './controllers';

const app: express.Application = express();
const port = process.env.PORT || 3000;
const debugLog: debug.IDebugger = debug('app');

const mongoServer = new MongoMemoryServer();
// const uri = await mongoServer.getUri();

export const DI = {} as {
  orm: MikroORM,
  em: EntityManager,
  playerRepository: EntityRepository<Player>,
  gameRepository: EntityRepository<Game>,
};

const loggerOptions: expressWinston.LoggerOptions = {
  transports: [new winston.transports.Console()],
  format: winston.format.combine(
    winston.format.json(),
    winston.format.prettyPrint(),
    winston.format.colorize({ all: true })
  ),
};

if (!process.env.DEBUG) {
  loggerOptions.meta = false; // when not debugging, make terse
}

(async () => {
  DI.orm = await MikroORM.init();
  DI.em = DI.orm.em;
  DI.playerRepository = DI.orm.em.getRepository(Player);
  DI.gameRepository = DI.orm.em.getRepository(Game);
  
  app.use(express.json())
  app.use(cors());
  app.use(expressWinston.logger(loggerOptions));
  app.use((req, res, next) => RequestContext.create(DI.orm.em, next));
  app.get('/', (req, res) => res.json({ message: "This is a game server for Power Grid: USA"}));
  app.use('/game', GameController);
  app.use('/player', PlayerController);
  app.use((req, res) => res.status(404).json({ message: 'No route found'}));

  app.listen(port, () => {
    console.log(`PowerGrid game server started at http://localhost:${port}`);
  });
})();
