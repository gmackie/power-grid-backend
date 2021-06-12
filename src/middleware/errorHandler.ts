import { Response } from "express";

export class ErrorHandler extends Error {
  statusCode: number;
  message: string;

  constructor(statusCode: number, message: string) {
    super();
    this.statusCode = statusCode;
    this.message = message;
  }
}

export const handleError = (error: ErrorHandler, response: Response) => {
  const { statusCode, message } = error;
  response.status(statusCode).json({
    status: 'error',
    success: false,
    statusCode,
    message,
  });
};