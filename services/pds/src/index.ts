if (!process.env.LOG_ENABLED && process.env.LOG_ENABLED !== "true") process.env["LOG_ENABLED"] = 'true';

import { PDS, AppContext, envToCfg, envToSecrets, readEnv, httpLogger, createLexiconServer } from "@atproto/pds";
// import { Options as XrpcServerOptions } from '@atproto/xrpc-server';
import { checkHandleRoute } from "./check_handle_route";
import pkg from "@atproto/pds/package.json";

async function main() {
  const env = getEnv();
  const cfg = envToCfg(env);
  const secrets = envToSecrets(env);
  // const ctx = await AppContext.fromConfig(cfg, secrets);
  const pds = await PDS.create(cfg, secrets);
  console.log('server:', pds.server);

  await pds.start();
  httpLogger.info("pds has started");
  // Graceful shutdown (see also https://aws.amazon.com/blogs/containers/graceful-shutdowns-with-ecs/)
  pds.app.get("/tls-check", (req, res) => {
    checkHandleRoute(pds, req, res);
  });
  process.on("SIGTERM", async () => {
    httpLogger.info("pds is stopping");
    await pds.destroy();
    httpLogger.info("pds is stopped");
  });
}

const getEnv = () => {
  const env = readEnv();
  env.version ||= pkg.version;
  env.port ||= 3000;
  return env;
};

// const createServer = (ctx: AppContext) => {
//   let server = createLexiconServer(serverOpts(ctx));
//   return API(server, ctx);
// };

// const serverOpts = (ctx: AppContext) => {
//   const xrpcOpts: XrpcServerOptions = {
//     validateResponse: false,
//     payload: {
//       jsonLimit: 150 * 1024, // 150kb
//       textLimit: 100 * 1024, // 100kb
//       blobLimit: ctx.cfg.service.blobUploadLimit,
//     },
//     catchall: proxyHandler(ctx),
//     rateLimits: ctx.ratelimitCreator
//       ? {
//         creator: ctx.ratelimitCreator,
//         global: [
//           {
//             name: 'global-ip',
//             durationMs: 5 * MINUTE,
//             points: 3000,
//           },
//         ],
//         shared: [
//           {
//             name: 'repo-write-hour',
//             durationMs: HOUR,
//             points: 5000, // creates=3, puts=2, deletes=1
//           },
//           {
//             name: 'repo-write-day',
//             durationMs: DAY,
//             points: 35000, // creates=3, puts=2, deletes=1
//           },
//         ],
//       }
//       : undefined,
//   };
//   return xrpcOpts;
// };

main();
