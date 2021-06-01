import { QueryOrder } from '@mikro-orm/core';
import { Request, Response } from 'express';
import Router from 'express-promise-router';
import { DI } from '../server';
import { Player } from '../entities';

const router = Router();

router.get('/', async (req: Request, res: Response) => {
    const players = await DI.playerRepository.findAll(['player'], { code: QueryOrder.DESC }, 20);
    res.json(players);
    // test
});

export const PlayerController = router;
