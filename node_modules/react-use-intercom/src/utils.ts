export const isSSR = typeof window === 'undefined';

/**
 * Removes object entries where the value equals to `undefined`
 *
 * @param obj
 */
export const removeUndefined = (obj: any) => {
  Object.keys(obj).forEach((key) => {
    if (obj[key] && typeof obj[key] === 'object') removeUndefined(obj[key]);
    else if (obj[key] === undefined) delete obj[key];
  });
  return obj;
};
