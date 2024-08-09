const path = require('path');

module.exports = () => ({
    name: "scroll-top-plugin",
    getClientModules() {
        return [path.resolve(__dirname, './plugin')];
    }
});
