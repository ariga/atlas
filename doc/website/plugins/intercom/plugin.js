import ExecutionEnvironment from "@docusaurus/ExecutionEnvironment";


export default (function () {
    if (!ExecutionEnvironment.canUseDOM) {
        return null;
    }
    window.intercomSettings = {
        api_base: "https://api-iam.intercom.io",
        app_id: "ugatw0l1"
    };
    (function(){var w=window;var ic=w.Intercom;if(typeof ic==="function"){ic('reattach_activator');ic('update',w.intercomSettings);}else{var d=document;var i=function(){i.c(arguments);};i.q=[];i.c=function(args){i.q.push(args);};w.Intercom=i;var l=function(){var s=d.createElement('script');s.type='text/javascript';s.async=true;s.src='https://widget.intercom.io/widget/ugatw0l1';var x=d.getElementsByTagName('script')[0];x.parentNode.insertBefore(s,x);};if(document.readyState==='complete'){l();}else if(w.attachEvent){w.attachEvent('onload',l);}else{w.addEventListener('load',l,false);}}})();
    return {};
})();
