import ExecutionEnvironment from "@docusaurus/ExecutionEnvironment";

export default (function () {
    if (!ExecutionEnvironment.canUseDOM) {
        return null;
    }
    return {
        onRouteDidUpdate({location, previousLocation}) {
            // Scroll to top when navigating to a new page.
            if (window.scrollY !== 0 && previousLocation != null && location.pathname !== previousLocation.pathname && location.hash === "") {
                window.scrollTo({
                    top: 0,
                    behavior: "instant",
                });
            }
        },
    };
})();