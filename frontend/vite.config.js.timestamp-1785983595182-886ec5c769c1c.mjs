import { defineConfig } from "file:///C:/ICONFIRM/frontend/node_modules/vite/dist/node/index.js";
import react from "file:///C:/ICONFIRM/frontend/node_modules/@vitejs/plugin-react/dist/index.js";
import tailwindcss from "file:///C:/ICONFIRM/frontend/node_modules/@tailwindcss/vite/dist/index.mjs";
import basicSsl from "file:///C:/ICONFIRM/frontend/node_modules/@vitejs/plugin-basic-ssl/dist/index.mjs";
var vite_config_default = defineConfig({
  plugins: [react(), tailwindcss(), basicSsl()],
  server: {
    host: "0.0.0.0",
    port: 9004,
    https: true,
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, "")
      }
    }
  }
});
export {
  vite_config_default as default
};
