import {
  DataAttributes,
  DataAttributesAvatar,
  DataAttributesCompany,
  IntercomProps,
  MessengerAttributes,
  RawDataAttributes,
  RawDataAttributesAvatar,
  RawDataAttributesCompany,
  RawIntercomProps,
  RawMessengerAttributes,
} from './types';
import { removeUndefined } from './utils';

export const mapMessengerAttributesToRawMessengerAttributes = (
  attributes: MessengerAttributes,
): RawMessengerAttributes => ({
  custom_launcher_selector: attributes.customLauncherSelector,
  alignment: attributes.alignment,
  vertical_padding: attributes.verticalPadding,
  horizontal_padding: attributes.horizontalPadding,
  hide_default_launcher: attributes.hideDefaultLauncher,
  session_duration: attributes.sessionDuration,
  action_color: attributes.actionColor,
  background_color: attributes.backgroundColor,
});

const mapDataAttributesCompanyToRawDataAttributesCompany = (
  attributes: DataAttributesCompany,
): RawDataAttributesCompany => ({
  company_id: attributes.companyId,
  name: attributes.name,
  created_at: attributes.createdAt,
  plan: attributes.plan,
  monthly_spend: attributes.monthlySpend,
  user_count: attributes.userCount,
  size: attributes.size,
  website: attributes.website,
  industry: attributes.industry,
  ...attributes.customAttributes,
});

const mapDataAttributesAvatarToRawDataAttributesAvatar = (
  attributes: DataAttributesAvatar,
): RawDataAttributesAvatar => ({
  type: attributes.type,
  image_url: attributes.imageUrl,
});

export const mapDataAttributesToRawDataAttributes = (
  attributes: DataAttributes,
): RawDataAttributes => ({
  email: attributes.email,
  user_id: attributes.userId,
  created_at: attributes.createdAt,
  name: attributes.name,
  phone: attributes.phone,
  last_request_at: attributes.lastRequestAt,
  unsubscribed_from_emails: attributes.unsubscribedFromEmails,
  language_override: attributes.languageOverride,
  utm_campaign: attributes.utmCampaign,
  utm_content: attributes.utmContent,
  utm_medium: attributes.utmMedium,
  utm_source: attributes.utmSource,
  utm_term: attributes.utmTerm,
  avatar:
    attributes.avatar &&
    mapDataAttributesAvatarToRawDataAttributesAvatar(attributes.avatar),
  user_hash: attributes.userHash,
  company:
    attributes.company &&
    mapDataAttributesCompanyToRawDataAttributesCompany(attributes.company),
  companies: attributes.companies?.map(
    mapDataAttributesCompanyToRawDataAttributesCompany,
  ),
  ...attributes.customAttributes,
});

export const mapIntercomPropsToRawIntercomProps = (
  props: IntercomProps,
): RawIntercomProps => {
  return removeUndefined({
    ...mapMessengerAttributesToRawMessengerAttributes(props),
    ...mapDataAttributesToRawDataAttributes(props),
  });
};
