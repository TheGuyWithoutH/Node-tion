export default {
  testEnvironment: "jsdom",
  transform: {
    "^.+\\.[tj]sx?$": "babel-jest",
    "^.+\\.mjs$": "babel-jest", // Add this line to handle .mjs files
  },
  transformIgnorePatterns: [
    "/node_modules/(?!prosemirror-highlight)/", // Allow transforming prosemirror-highlight
  ],
  setupFilesAfterEnv: ["<rootDir>/jest.setup.ts"],
  moduleNameMapper: {
    "^uuid$": "uuid",
    "^oniguruma-to-es$":
      "<rootDir>/node_modules/oniguruma-to-es/dist/index.mjs",
    "^@/(.*)$": "<rootDir>/src/$1",
  },
  coveragePathIgnorePatterns: ["/node_modules/", "/wailsjs/"],
};
