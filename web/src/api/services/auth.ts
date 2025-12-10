import { typedClient } from "../typedClient";

export const authService = {
  login: (request: { username: string; password: string }) =>
    typedClient.login(request),

  register: (request: {
    username: string;
    email: string;
    password: string;
    fullName: string;
  }) => typedClient.register(request),

  refreshToken: (refreshToken: string) =>
    typedClient.refreshToken({ refreshToken }),

  logout: () => typedClient.logout(),

  getProfile: () => typedClient.getProfile(),

  validateToken: (token: string) => typedClient.validateToken({ token }),
};
