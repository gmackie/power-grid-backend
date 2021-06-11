import dotenv from 'dotenv';

dotenv.config();

const getDefault = (value: any, defaultValue: any) => {
  if (!value || value === undefined) {
    return defaultValue;
  }
  return value;
};

export const config = {
  DB_URL: getDefault(process.env.DB_URL, 'mongodb://localhost:27017/power-grid'),
  JWT_SECRET: getDefault(process.env.JWT_SECRET, 'REDACTED'),
  API_PORT: process.env.API_PORT ? Number.parseInt(process.env.API_PORT, 10) : 8080,
  SOCKET_PORT: process.env.SOCKET_PORT ? Number.parseInt(process.env.SOCKET_PORT, 10) : 65080,
  REDIS_PORT: process.env.REDIS_PORT ? Number.parseInt(process.env.REDIS_PORT, 10) : 6379,
  REDIS_HOST: getDefault(process.env.REDIS_HOST, 'localhost'),

  SALT_ROUNDS: process.env.SALT_ROUNDS ? Number.parseInt(process.env.SALT_ROUNDS, 10) : 6,
};