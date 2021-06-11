import { NextFunction, Request, Response } from 'express';
import jwt from 'jsonwebtoken';
import { config } from '../config';

const authenticated = (request: Request, response: Response, next: NextFunction) => {
  const token = request.headers.authorization;
  jwt.verify(token, config.JWT_SECRET, (error, _) => {
    if (error) {
      response.json('Token not provided');
    } else {
      next();
    }
  });
};

export const verifyToken = (token: string) => {
  try {
    const decoded = jwt.verify(token, config.JWT_SECRET);
    return decoded;
  } catch {
    return false;
  }
};

export default authenticated;