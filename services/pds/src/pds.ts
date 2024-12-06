import 'express-async-errors';

import { AppContext } from "@atproto/pds";
import express from "express";

export class PDS {
  public ctx: AppContext;
  public app: express.Application;

  constructor(opts: { ctx: AppContext; app: express.Application; }) {
    this.ctx = opts.ctx;
    this.app = opts.app;
  }
}
