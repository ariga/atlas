import * as React from 'react';

import { IntercomContextValues } from './types';

const IntercomContext = React.createContext<IntercomContextValues | undefined>(
  undefined,
);

export default IntercomContext;
