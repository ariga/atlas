import ExecutionEnvironment from "@docusaurus/ExecutionEnvironment";

export default (function () {
  if (!ExecutionEnvironment.canUseDOM) {
    return null;
  }

  return {
    onRouteDidUpdate() {
      function scrollToTop() {
        if (window.scrollY !== 0) {
          window.scrollTo({
            top: 0,
            behavior: "instant",
          });
        }
      }

      setTimeout(scrollToTop, 100);
    },
  };
})();
