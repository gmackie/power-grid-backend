import { QueryOrder } from '@mikro-orm/core';
import { Request, Response } from 'express';
import Router from 'express-promise-router';
import { DI } from '../server';
import { Game } from '../entities';

const router = Router();

router.get('/', async (req: Request, res: Response) => {
    const games = await DI.gameRepository.findAll(['game'], { code: QueryOrder.DESC }, 20);
    res.json(games);
});

export const GameController = router;