import Image from "next/image";
import { Icon } from "./ui";
import { homeAssets } from "../../shared/header";

type Integration = { name: string; logo: string } | { name: string; icon: string };

export const dataSources: Integration[] = [
  { name: "PDF / Office", icon: "file" },
  { name: "Feishu", logo: "feishu.ico" },
  { name: "Confluence", logo: "confluence.svg" },
  { name: "DingTalk Dokümanları", logo: "dingtalk.svg" },
  { name: "GitLab", logo: "gitlab.png" },
  { name: "Tencent IMA", logo: "ima.png" },
  { name: "Notion", logo: "notion.ico" },
  { name: "Yuque", logo: "yuque.ico" },
  { name: "RSS", logo: "rss.svg" },
];

export const clients: Integration[] = [
  { name: "Slack", logo: "slack.svg" },
  { name: "Telegram", logo: "telegram.svg" },
  { name: "Chrome", logo: "chrome.svg" },
  { name: "BrowserSkill", logo: "browserskill.png" },
  { name: "MCP Server", logo: "mcp.svg" },
  { name: "API", icon: "code" },
];

export const modelProviders: Integration[] = [
  { name: "OpenAI", logo: "openai.svg" },
  { name: "Claude", logo: "anthropic.svg" },
  { name: "Gemini", logo: "gemini-color.svg" },
  { name: "DeepSeek", logo: "deepseek-color.svg" },
  { name: "Qwen", logo: "qwen-color.svg" },
  { name: "Hunyuan", logo: "hunyuan-color.svg" },
  { name: "Zhipu", logo: "zhipu.svg" },
  { name: "Kimi", logo: "moonshot.svg" },
  { name: "Volcano Engine", logo: "volcengine.svg" },
  { name: "MiniMax", logo: "minimax.svg" },
  { name: "LiteLLM", logo: "litellm.ico" },
];

export function IntegrationMark({ item }: { item: Integration }) {
  return <>
    {"logo" in item
      ? <Image src={`${homeAssets}/brands/${item.logo}`} alt="" aria-hidden="true" width={22} height={22} />
      : <Icon name={item.icon} />}
    {item.name}
  </>;
}
