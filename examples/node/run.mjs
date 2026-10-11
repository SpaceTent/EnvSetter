import { spawn } from "node:child_process";

const command = process.argv.slice(2);
if (command.length === 0) {
  console.error("usage: node run.mjs command [args...]");
  process.exit(2);
}

const child = spawn("envsetter", ["run", "--", ...command], {
  env: process.env,
  stdio: "inherit",
});

child.on("error", (error) => {
  console.error(`unable to start envsetter: ${error.message}`);
  process.exit(1);
});

child.on("exit", (code, signal) => {
  if (signal) {
    process.kill(process.pid, signal);
  } else {
    process.exit(code ?? 1);
  }
});
