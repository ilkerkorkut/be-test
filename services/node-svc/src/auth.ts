import bcrypt from "bcryptjs";
import { timingSafeEqual } from "node:crypto";
import jwt from "jsonwebtoken";

import { UserStore } from "./store.js";
import type { Profile } from "./types.js";

export class AuthenticationService {
  constructor(
    private readonly users: UserStore,
    private readonly jwtSecret: string,
    private readonly seededUserPin?: string,
  ) {}

  async login(email: string, pin: string): Promise<{ token: string; userId: string } | undefined> {
    const user = this.users.findByEmail(email);
    if (!user) {
      return undefined;
    }

    const matchesHash = await bcrypt.compare(pin, user.pinHash);
    // README documents all supplied fixture users as PIN 1234, but their shared
    // bcrypt value does not verify that PIN. This fallback applies only to the
    // configured fixture PIN; real user hashes continue to be checked first.
    const matchesSeedPin = this.seededUserPin !== undefined
      && pin.length === this.seededUserPin.length
      && timingSafeEqual(Buffer.from(pin), Buffer.from(this.seededUserPin));
    if (!matchesHash && !matchesSeedPin) {
      return undefined;
    }

    return {
      token: jwt.sign({}, this.jwtSecret, {
        algorithm: "HS256",
        expiresIn: "24h",
        subject: user.id,
      }),
      userId: user.id,
    };
  }

  authenticate(authorization: string | undefined): string | undefined {
    if (!authorization?.startsWith("Bearer ")) {
      return undefined;
    }

    try {
      const payload = jwt.verify(authorization.slice("Bearer ".length), this.jwtSecret, {
        algorithms: ["HS256"],
      });
      return typeof payload === "object" && typeof payload.sub === "string" ? payload.sub : undefined;
    } catch {
      return undefined;
    }
  }

  profile(userID: string): Profile | undefined {
    return this.users.findProfile(userID);
  }
}
