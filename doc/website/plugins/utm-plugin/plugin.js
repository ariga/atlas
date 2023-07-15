import ExecutionEnvironment from "@docusaurus/ExecutionEnvironment";

const CONFIG = {
  selector: 'a[href*="atlasgo.cloud"]',
  utmSource: "atlasgo",
  utmMedium: "organic",
};

export default (function () {
  if (!ExecutionEnvironment.canUseDOM) {
    return null;
  }
  return {
    onRouteDidUpdate({ location: { pathname } }) {
      const links = document.querySelectorAll(CONFIG.selector);

      links.forEach(function (link) {
        const url = new URL(link.href);

        const locationPath = pathname.split("/").slice(1).join("_");

        url.searchParams.set("utm_source", CONFIG.utmSource);
        url.searchParams.set("utm_medium", CONFIG.utmMedium);
        url.searchParams.set("utm_term", locationPath || "main");

        link.href = url.toString();
      });
    },
  };
})();
