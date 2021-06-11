import { Request, Response, NextFunction } from 'express';

export default class AuthController {
  async login(request: Request, response: Response, next: NextFunction) {
    try {
      const { username, password, reserved } = request.body;

      if (!username) {
        throw new ErrorHandler(401, 'No username provided');
      }

      if (!reserved && username) {
        const check = await checkExisting(username);
        if (check) {
          throw new ErrorHandler(401, 'Username is already in use');
        }

        const token = jwt.sign({ username, reserved: false }, JWT_SECRET);
        return response.json({ success: true, token });
      }

      const user = await Users.findOne({ username });
      const match = await bcrypt.compare(password, user.password);

      if (user && match) {
        const token = jwt.sign({ username: user.username }, JWT_SECRET);
        return response.json({ success: true, token });
      }
      throw new ErrorHandler(401, 'Username or password is incorrect');
    } catch (error) {
      next(error);
    }
  }

  async register(request: Request, response: Response, next: NextFunction) {
    try {
      if (!request.body) {
        throw new ErrorHandler(400, 'Invalid Request');
      }

      const { username, password } = request.body;
      const check = await checkExisting(username);

      if (check) {
        throw new ErrorHandler(400, 'Username already exists');
      }

      const hash = bcrypt.hashSync(password, SALT_ROUNDS);
      const newUser = new Users({ username, password: hash });

      await newUser.save();

      return response.json({
        success: true,
        message: 'Successfully registered',
      });
    } catch (error) {
      next(error);
    }
  }

  async verify(request: Request, response: Response) {
    if (!request.body) {
      throw new ErrorHandler(401, 'Unauthorized user and/or route');
    }

    const { token } = request.body;
    const decoded = verifyToken(token);

    if (!decoded) {
      throw new ErrorHandler(401, 'Unauthorized action. JWT expired');
    }

    return response.json({ success: true, decoded });
  }

  async geExternalIp(request: Request, response: Response) {
    const ip = await getExternalIp();

    return response.json({ success: true, ip });
  }

}