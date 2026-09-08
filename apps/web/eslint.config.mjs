import next from "eslint-config-next";

// `npm run lint` ran `eslint .` with no config file, which ESLint 9 treats as a
// hard error, so linting was never actually possible.
const config = [
  { ignores: [".next/**", "node_modules/**", "next-env.d.ts"] },
  ...next
];

export default config;
