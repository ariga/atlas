const path = require('path');

module.exports = () => ({
    name: "intercom-plugin",
    getClientModules() {
        return [path.resolve(__dirname, './plugin')];
    }
});
