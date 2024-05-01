const path = require('path');

module.exports = () => ({
  name: "atlas-page-view-plugin",
  getClientModules() {
    return [path.resolve(__dirname, './plugin')];
  }
});
