import * as logger from './logger';
import { IntercomMethod } from './types';
import { isSSR } from './utils';

/**
 * Safely exposes `window.Intercom` and passes the arguments to the instance
 *
 * @param method method passed to the `window.Intercom` instance
 * @param args arguments passed to the `window.Intercom` instance
 *
 * @see {@link https://developers.intercom.com/installing-intercom/docs/intercom-javascript}
 */
const IntercomAPI = (method: IntercomMethod, ...args: Array<any>) => {
  if (!isSSR && window.Intercom) {
    return window.Intercom.apply(null, [method, ...args]);
  } else {
    logger.log('error', `${method} Intercom instance is not initalized yet`);
  }
};

export default IntercomAPI;
