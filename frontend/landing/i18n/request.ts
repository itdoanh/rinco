import { getRequestConfig } from 'next-intl/server';

// Minimal i18n request stub for next-intl plugin compatibility.
// The landing app currently uses a single locale (vi_VN) defined in metadata.
export default getRequestConfig(async () => ({
  locale: 'vi',
  messages: {},
}));
