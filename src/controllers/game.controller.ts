import { QueryOrder, wrap } from '@mikro-orm/core';
import { Request, Response } from 'express';
import Router from 'express-promise-router';
import { DI } from '../server';
import { Game, Player } from '../entities';

function generateRandomNumber(numberOfCharacters: number) {
   let randomValues = '';
   const stringValues = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ';  
   const sizeOfCharacter = stringValues.length;  
for (var i = 0; i < numberOfCharacters; i++) {
      randomValues = randomValues+stringValues.charAt(Math.floor(Math.random() * sizeOfCharacter));
   }
   return randomValues;
} 

const router = Router();

router.get('/', async (req: Request, res: Response) => {
    const games = await DI.gameRepository.findAll(['players'], { code: QueryOrder.DESC }, 20);
    res.json(games);
});

router.post('/', async (req: Request, res: Response) => {
  if (!req.body.host) {
    res.status(400);
    return res.json({ message: '`host` is missing' });
  }

  try {
    const { host } = req.body;
    const code = generateRandomNumber(5);
    const game = new Game(code, host);
    wrap(game).assign(req.body);
    await DI.gameRepository.persist(game).flush();
    res.json(game);
  } catch(e) {
    return res.status(400).json({ message: e.message});
  }
}); 

router.get('/:code', async (req: Request, res: Response) => {
  try {
    const game = await DI.gameRepository.findOne({ code: req.params.code }, ['players']);
    if (!game) {
      return res.status(404).json({ message: 'game not found' });
    }

    res.json(game);
  } catch(e) {
    return res.status(400).json({ message: e.message });
  }
});

router.post('/:code/add_player', async (req: Request, res: Response) => {
  if (!req.body.name) {
    res.status(400).json({ message: 'missing player name' });
  }

  try {
    const game = await DI.gameRepository.findOne({ code: req.params.code }, ['players']);
    if (!game) {
      return res.status(404).json({ message: 'game not found' });
    }
    const player = new Player(req.body.name, game);
    game.players.add(player);
    await DI.gameRepository.flush();

    res.json(game);
  } catch(e) {
    return res.status(400).json({ message: e.message });
  }
});

export const GameController = router;