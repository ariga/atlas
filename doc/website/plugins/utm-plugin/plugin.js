import ExecutionEnvironment from "@docusaurus/ExecutionEnvironment";

const CONFIG = {
  selector: 'a[href*="atlasgo.cloud"]',
  utmSource: "atlasgo",
  utmMedium: "website",
};

// Correct link: host/example?utm_term=custom#hash

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

        if (!url.searchParams.has("utm_source")) {
          url.searchParams.set("utm_source", CONFIG.utmSource);
        }

        if (!url.searchParams.has("utm_medium")) {
          url.searchParams.set("utm_medium", CONFIG.utmMedium);
        }

        if (!url.searchParams.has("utm_term")) {
          url.searchParams.set("utm_term", locationPath || "main");
        }

        link.href = url.toString();
      });
    },
  };
})();
