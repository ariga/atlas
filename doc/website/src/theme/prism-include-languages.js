/**
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */
import ExecutionEnvironment from '@docusaurus/ExecutionEnvironment';
import siteConfig from '@generated/docusaurus.config';

const prismIncludeLanguages = (Prism) => {
  Prism.languages.applylog = {
    'version': /\d{14}/,
    'duration': /\b[\d\\.]+(s|ms|µs|m)/,
    'action1': /\s\s+-{2}\s/,
    'action2': /\s\s+-{25}/,
    'action3': /\s\s+->\s/,
    'error': /(Error:\s.+|\s+.+(assertions failed:|check assertion)\s.+)/i,
  };
  if (ExecutionEnvironment.canUseDOM) {
    const {
      themeConfig: {prism: {additionalLanguages = []} = {}},
    } = siteConfig;
    window.Prism = Prism;
    additionalLanguages.forEach((lang) => {
      require(`prismjs/components/prism-${lang}`); // eslint-disable-line
    });
    delete window.Prism;
  }
};

export default prismIncludeLanguages;
