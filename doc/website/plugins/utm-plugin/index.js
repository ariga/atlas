const path = require('path');

module.exports = () => ({
  name: "atlas-utm-plugin",
  getClientModules() {
    return [path.resolve(__dirname, './plugin')];
  }
});
