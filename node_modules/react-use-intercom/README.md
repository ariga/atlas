<p align="center"><img src="https://raw.githubusercontent.com/devrnt/react-use-intercom/main/assets/logo.png" alt="Logo" height="120px" style="margin-top: 20px;"/></p>
<h1 align="center">react-use-intercom</h1>
<p align="center">A React <a href="https://www.intercom.com" alt="Intercom">Intercom </a> integration powered by hooks.</p>

<p align="center">
<img alt="ci" src="https://github.com/devrnt/react-use-intercom/workflows/CI/badge.svg?branch=main">
<img alt="version" src="https://img.shields.io/npm/v/react-use-intercom.svg" />
<img alt="downloads" src="https://badgen.net/npm/dw/react-use-intercom" />
<img alt="minzipped size" src="https://badgen.net/bundlephobia/minzip/react-use-intercom">
<img alt="known vulnerabilities" src="https://snyk.io/test/github/devrnt/react-use-intercom/badge.svg">
</p>

## Features
* Hooks
* Written in TypeScript
* Documented, self explaining methods 
* [Tiny size](https://bundlephobia.com/result?p=react-use-intercom@latest) without any external libraries
* Safeguard for SSR environments (NextJS, Gatsby)
* Compatible to hook into existing Intercom instance (loaded by [Segment](https://segment.com/))

## Installation

```
yarn add react-use-intercom
```

## Quickstart

```js
import * as React from 'react';

import { IntercomProvider, useIntercom } from 'react-use-intercom';

const INTERCOM_APP_ID = 'your-intercom-app-id';

const App = () => (
  <IntercomProvider appId={INTERCOM_APP_ID}>
    <HomePage />
  </IntercomProvider>
);

// Anywhere in your app
const HomePage = () => {
  const { boot, shutdown, hide, show, update } = useIntercom();

  return <button onClick={boot}>Boot intercom! ☎️</button>;
};
```

## Context
This library is a React abstraction of [IntercomJS](https://developers.intercom.com/installing-intercom/docs/intercom-for-web). `react-use-intercom` tries to keep as close as a one-on-one abstraction of the "vanilla" Intercom functionality.
 
Note that a lot of issues could be related to the vanilla IntercomJS. Please see https://forum.intercom.com/s/ before reporting an issue here.

## Links
* [API](#api)
* [Playground](#playground)
* [Examples](#examples)
* [TypeScript](#typescript)
* [Troubleshoot](#troubleshoot)
* [Advanced](#advanced)

## API
* [IntercomProvider](#intercomprovider)
* [useIntercom](#useintercom)
* [IntercomProps](#intercomprops)

### IntercomProvider 
`IntercomProvider` is used to initialize the `window.Intercom` instance. It makes sure the initialization is only done once. If any listeners are passed, the `IntercomProvider` will make sure these are attached.

Place the `IntercomProvider` as high as possible in your application. This will make sure you can call `useIntercom` anywhere.

#### Props
| name                | type             | description                                                                             | required | default |
|---------------------|------------------|-----------------------------------------------------------------------------------------|----------|---------|
| appId               | string           | app ID of your Intercom instance                                                        | true     |         |
| children            | React.ReactNode  | React children                                                                          | true     |         |
| autoBoot            | boolean          | indicates if Intercom should be automatically booted. If `true` no need to call `boot`, the `IntercomProvider` will call it for you  | false    |   false |
| onHide              | () => void       | triggered when the Messenger hides                                                      | false    |         |
| onShow              | () => void       | triggered when the Messenger shows                                                      | false    |         |
| onUnreadCountChange | (number) => void | triggered when the current number of unread messages changes                            | false    |         |
| onUserEmailSupplied | () => void | triggered when a visitor enters their email into the Messenger                         | false    |         |
| shouldInitialize    | boolean | indicates if the Intercom should be initialized. Can be used in multistaged environment          | false    | true    |
| apiBase    | string | If you need to route your Messenger requests through a different endpoint than the default. Generally speaking, this is not needed.<br/> Format: `https://${INTERCOM_APP_ID}.intercom-messenger.com` (See: [https://github.com/devrnt/react-use-intercom/pull/96](https://github.com/devrnt/react-use-intercom/pull/96))         | false    |         |
| initializeDelay | number | Indicates if the intercom initialization should be delayed, delay is in ms, defaults to 0. See https://github.com/devrnt/react-use-intercom/pull/236 | false    |         |
| autoBootProps | IntercomProps | Pass properties to `boot` method when `autoBoot` is `true` | false    |         |

#### Example
```javascript
const App = () => {
  const [unreadMessagesCount, setUnreadMessagesCount] = React.useState(0);

  const onHide = () => console.log('Intercom did hide the Messenger');
  const onShow = () => console.log('Intercom did show the Messenger');
  const onUnreadCountChange = (amount: number) => {
    console.log('Intercom has a new unread message');
    setUnreadMessagesCount(amount);
  };
  const onUserEmailSupplied = () => {
    console.log('Visitor has entered email');
  };

  return (
    <IntercomProvider
      appId={INTERCOM_APP_ID}
      onHide={onHide}
      onShow={onShow}
      onUnreadCountChange={onUnreadCountChange}
      onUserEmailSupplied={onUserEmailSupplied}
      autoBoot
    >
      <p>Hi there, I am a child of the IntercomProvider</p>
    </IntercomProvider>
  );
};
```

### useIntercom
Used to retrieve all methods bundled with Intercom. These are based on the official [Intercom docs](https://developers.intercom.com/installing-intercom/docs/javascript-api-attributes-objects). Some extra methods were added to improve convenience.

 Make sure `IntercomProvider` is wrapped around your component when calling `useIntercom()`. 

**Remark** - You can't use `useIntercom()` in the same component where `IntercomProvider` is initialized.

#### API

| name            | type                                       | description                                                                                                                         |
|-----------------|--------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------|
| isOpen              | boolean          | the visibility status of the messenger                                                  |
| boot            | (props?: IntercomProps) => void            | boots the Intercom instance, not needed if `autoBoot` in `IntercomProvider` is `true`                                               |
| shutdown        | () => void                                 | shuts down the Intercom instance                                                                                                     |
| hardShutdown    | () => void                                 | same functionality as `shutdown`, but makes sure the Intercom cookies, `window.Intercom` and `window.intercomSettings` are removed. |
| update          | (props?: IntercomProps) => void            | updates the Intercom instance with the supplied props. To initiate a 'ping', call `update` without props                            |
| hide            | () => void                                 | hides the Messenger, will call `onHide` if supplied to `IntercomProvider`                                                           |
| show            | () => void                                 | shows the Messenger, will call `onShow` if supplied to `IntercomProvider`                                                           |
| showMessages    | () => void                                 | shows the Messenger with the message list                                                                                           |
| showNewMessage  | (content?: string) => void                 | shows the Messenger as if a new conversation was just created. If `content` is passed, it will fill in the message composer         |
| getVisitorId    | () => string                               | gets the visitor id                                                                                                                 |
| startTour       | (tourId: number) => void                   | starts a tour based on the `tourId`                                                                                                 |
| trackEvent      | (event: string, metaData?: object) => void | submits an `event` with optional `metaData`      
| showArticle      | (articleId: string) => void | opens the Messenger with the specified article by `articleId`
| startSurvey      | (surveyId: number) => void | Trigger a survey in the Messenger by `surveyId`

#### Example
```javascript
import * as React from 'react';

import { IntercomProvider, useIntercom } from 'react-use-intercom';

const INTERCOM_APP_ID = 'your-intercom-app-id';

const App = () => (
  <IntercomProvider appId={INTERCOM_APP_ID}>
    <HomePage />
  </IntercomProvider>
);

const HomePage = () => {
  const {
    boot,
    shutdown,
    hardShutdown,
    update,
    hide,
    show,
    showMessages,
    showNewMessage,
    getVisitorId,
    startTour,
    trackEvent,
    showArticle,
    startSurvey
  } = useIntercom();

  const bootWithProps = () => boot({ name: 'Russo' });
  const updateWithProps = () => update({ name: 'Ossur' });
  const handleNewMessages = () => showNewMessage();
  const handleNewMessagesWithContent = () => showNewMessage('content');
  const handleGetVisitorId = () => console.log(getVisitorId());
  const handleStartTour = () => startTour(123);
  const handleTrackEvent = () => trackEvent('invited-friend');
  const handleTrackEventWithMetaData = () =>
    trackEvent('invited-frind', {
      name: 'Russo',
    });
  const handleShowArticle = () => showArticle(123456);
  const handleStartSurvey = () => startSurvey(123456);

  return (
    <>
      <button onClick={boot}>Boot intercom</button>
      <button onClick={bootWithProps}>Boot with props</button>
      <button onClick={shutdown}>Shutdown</button>
      <button onClick={hardShutdown}>Hard shutdown</button>
      <button onClick={update}>Update clean session</button>
      <button onClick={updateWithProps}>Update session with props</button>
      <button onClick={show}>Show messages</button>
      <button onClick={hide}>Hide messages</button>
      <button onClick={showMessages}>Show message list</button>
      <button onClick={handleNewMessages}>Show new messages</button>
      <button onClick={handleNewMessagesWithContent}>
        Show new message with pre-filled content
      </button>
      <button onClick={handleGetVisitorId}>Get visitor id</button>
      <button onClick={handleStartTour}>Start tour</button>
      <button onClick={handleTrackEvent}>Track event</button>
      <button onClick={handleTrackEventWithMetaData}>
        Track event with metadata
      </button>
      <button onClick={handleShowArticle}>Open article in Messenger</button>
      <button onClick={handleStartSurvey}>Start survey in Messenger</button>
    </>
  );
};
``` 
### IntercomProps 
All the Intercom default attributes/props are camel cased (`appId` instead of `app_id`) in `react-use-intercom`, see [IntercomProps](src/types.ts) to see what attributes you can pass to `boot` or `update`. Or check the Intercom [docs](https://developers.intercom.com/installing-intercom/docs/javascript-api-attributes-objects)
 to see all the available attributes/props.

 **Remark** - all the listed Intercom attributes [here](https://developers.intercom.com/installing-intercom/docs/javascript-api-attributes-objects) are snake cased, in `react-use-intercom` these are camel cased.

 #### Custom attributes
 Still want to pass custom attributes to Intercom? Whether `boot` or `update` is used, you can add your custom properties by passing these through `customAttributes` in the `boot` or `update` method. 

**Remark** - the keys of the `customAttributes` object should be snake cased (this is how Intercom wants them). They are rawly passed to Intercom.
 ```javascript
 const { boot } = useIntercom();

 boot({ 
  name: 'Russo',
  customAttributes: { custom_attribute_key: 'hi there' },
})
 ```

## Playground
Small playground to showcase the functionalities of `react-use-intercom`. 

### useIntercom
[https://devrnt.github.io/react-use-intercom/#/useIntercom](https://devrnt.github.io/react-use-intercom/#/useIntercom)

### useIntercom (with Intercom tour)
[https://devrnt.github.io/react-use-intercom/#/useIntercomTour](https://devrnt.github.io/react-use-intercom/#/useIntercomTour)

## Examples 
Go to [examples](https://github.com/devrnt/react-use-intercom/tree/main/apps/examples) to check out some integrations (Gatsby, NextJS...).

## TypeScript
All the possible pre-defined options to pass to the Intercom instance are typed. So whenever you have to pass [IntercomProps](src/types.ts), all the possible properties will be available out of the box.
These props are `JavaScript` 'friendly', so [camelCase](https://en.wikipedia.org/wiki/Camel_case). No need to pass the props with [snake_cased](https://en.wikipedia.org/wiki/Snake_case) keys.

**Remark** - if you want to pass custom properties, you should still use [snake_cased](https://en.wikipedia.org/wiki/Snake_case) keys.


## Troubleshoot
* I'm seeing `Please wrap your component with IntercomProvider` in the console.
> Make sure `IntercomProvider` is initialized before calling `useIntercom()`. You only need to initialize `IntercomProvider` once. It is advised to initialize `IntercomProvider` as high as possible in your application tree.

> Make sure you aren't calling `useIntercom()` in the same component where you initialized `IntercomProvider`.

* I'm seeing `Some invalid props were passed to IntercomProvider. Please check following props: [properties]` in the console.
> Make sure you're passing the correct properties to the `IntercomProvider`. Check [IntercomProvider](#intercomprovider) to see all the properties.
> Mind that all the properties in `react-use-intercom` are camel cased, except for the `customAttributes` property in the `boot` and `update` method from `useIntercom`.

## Advanced

### Delay initialization

`<IntercomProvider />` uses an official intercom snippet and is directly initialized on load. In the background this snippet will load some external code that makes Intercom work. All of this magic happens on the initial load and in some use cases this can become problematic (E.g. when LCP is priority).

Since [v1.2.0](https://github.com/devrnt/react-use-intercom/releases/tag/v1.2.0) it's possible to delay this initialisation by passing `initializeDelay` in `<IntercomProvider />` (it's in milliseconds). However most of the users won't need to mess with this.

For reference see https://github.com/devrnt/react-use-intercom/pull/236 and https://forum.intercom.com/s/question/0D52G00004WxWLs/can-i-delay-loading-intercom-on-my-site-to-reduce-the-js-load
