import type { SettingsGrouped } from "@/shared/api/settings.types";

export type WebSearchSettingsFieldType = "int" | "bool" | "string" | "password" | "textarea" | "select";

export type WebSearchSettingsField = {
  namespace: "websearch";
  key:
    | "web_search_enable"
    | "web_search_provider"
    | "searxng_base_url"
    | "tavily_api_key"
    | "web_search_max_results"
    | "web_search_timeout_seconds"
    | "web_fetch_enable"
    | "web_fetch_max_chars"
    | "web_search_prompt";
  labelKey: string;
  descriptionKey: string;
  type: WebSearchSettingsFieldType;
  placeholder?: string;
  placeholderKey?: string;
  optionKeys?: { value: string; labelKey: string }[];
};

export const WEB_SEARCH_SETTINGS_FIELDS: WebSearchSettingsField[] = [
  {
    namespace: "websearch",
    key: "web_search_enable",
    labelKey: "webSearchEnable.label",
    descriptionKey: "webSearchEnable.description",
    type: "bool",
  },
  {
    namespace: "websearch",
    key: "web_search_provider",
    labelKey: "webSearchProvider.label",
    descriptionKey: "webSearchProvider.description",
    type: "select",
    optionKeys: [
      { value: "searxng", labelKey: "webSearchProvider.options.searxng" },
      { value: "tavily", labelKey: "webSearchProvider.options.tavily" },
    ],
  },
  {
    namespace: "websearch",
    key: "searxng_base_url",
    labelKey: "searxngBaseURL.label",
    descriptionKey: "searxngBaseURL.description",
    type: "string",
    placeholder: "http://searxng:8080",
  },
  {
    namespace: "websearch",
    key: "tavily_api_key",
    labelKey: "tavilyAPIKey.label",
    descriptionKey: "tavilyAPIKey.description",
    type: "password",
    placeholder: "tvly-...",
  },
  {
    namespace: "websearch",
    key: "web_search_max_results",
    labelKey: "webSearchMaxResults.label",
    descriptionKey: "webSearchMaxResults.description",
    type: "int",
    placeholder: "5",
  },
  {
    namespace: "websearch",
    key: "web_search_timeout_seconds",
    labelKey: "webSearchTimeout.label",
    descriptionKey: "webSearchTimeout.description",
    type: "int",
    placeholder: "15",
  },
  {
    namespace: "websearch",
    key: "web_fetch_enable",
    labelKey: "webFetchEnable.label",
    descriptionKey: "webFetchEnable.description",
    type: "bool",
  },
  {
    namespace: "websearch",
    key: "web_fetch_max_chars",
    labelKey: "webFetchMaxChars.label",
    descriptionKey: "webFetchMaxChars.description",
    type: "int",
    placeholder: "8000",
  },
  {
    namespace: "websearch",
    key: "web_search_prompt",
    labelKey: "webSearchPrompt.label",
    descriptionKey: "webSearchPrompt.description",
    type: "textarea",
    placeholderKey: "defaultPromptPlaceholder",
  },
];

export function webSearchFieldID(field: WebSearchSettingsField): string {
  return `${field.namespace}.${field.key}`;
}

export function flattenWebSearchSettings(grouped: SettingsGrouped): Record<string, string> {
  const result: Record<string, string> = {};
  for (const item of grouped.websearch ?? []) {
    result[`websearch.${item.key}`] = item.value ?? "";
  }
  return applyWebSearchSettingsDefaults(result);
}

export function applyWebSearchSettingsDefaults(settings: Record<string, string>): Record<string, string> {
  return {
    ...settings,
    "websearch.web_search_enable": settings["websearch.web_search_enable"] || "false",
    "websearch.web_search_provider": settings["websearch.web_search_provider"] || "searxng",
    "websearch.searxng_base_url": settings["websearch.searxng_base_url"] ?? "",
    "websearch.tavily_api_key": settings["websearch.tavily_api_key"] ?? "",
    "websearch.web_search_max_results": settings["websearch.web_search_max_results"] || "5",
    "websearch.web_search_timeout_seconds": settings["websearch.web_search_timeout_seconds"] || "15",
    "websearch.web_fetch_enable": settings["websearch.web_fetch_enable"] || "true",
    "websearch.web_fetch_max_chars": settings["websearch.web_fetch_max_chars"] || "8000",
    "websearch.web_search_prompt": settings["websearch.web_search_prompt"] ?? "",
  };
}

export function toWebSearchEditorField(
  field: WebSearchSettingsField,
  translate: (key: string) => string,
) {
  return {
    id: webSearchFieldID(field),
    label: translate(field.labelKey),
    description: translate(field.descriptionKey),
    type: field.type,
    placeholder: field.placeholderKey ? translate(field.placeholderKey) : field.placeholder,
    options: field.optionKeys?.map((option) => ({
      value: option.value,
      label: translate(option.labelKey),
    })),
  } as const;
}
