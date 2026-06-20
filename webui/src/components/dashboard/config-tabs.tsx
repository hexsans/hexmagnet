import { useNavigate, useParams, } from "react-router";
import { useTranslation, } from "react-i18next";
import { Tabs, TabsContent, TabsList, TabsTrigger, } from "@/components/ui/tabs";
import { ServerConfigForm, } from "@/components/dashboard/server-config-form";
import { ClassifierLLMConfigForm, } from "@/components/dashboard/classifier-llm-config-form";
import { StorageConfigForm, } from "@/components/dashboard/storage-config-form";
import { DHTConfigForm, } from "@/components/dashboard/dht-config-form";

export function ConfigTabs() {
  const { t, } = useTranslation();
  const navigate = useNavigate();
  const { configTab = "general", } = useParams();

  return (
    <Tabs value={configTab} onValueChange={(v,) => navigate(`/dashboard/config/${v}`,)} className="w-full">
      <TabsList className="font-mono bg-card border border-border h-9 gap-0.5 p-1">
        <TabsTrigger value="general" className="gap-1.5 text-xs data-active:bg-green/10 data-active:text-green">
          {t("dashboard.general",)}
        </TabsTrigger>
        <TabsTrigger value="dht" className="gap-1.5 text-xs data-active:bg-green/10 data-active:text-green">
          {t("dashboard.dht",)}
        </TabsTrigger>
        <TabsTrigger value="classifier" className="gap-1.5 text-xs data-active:bg-green/10 data-active:text-green">
          {t("dashboard.classifier",)}
        </TabsTrigger>
        <TabsTrigger value="storage" className="gap-1.5 text-xs data-active:bg-green/10 data-active:text-green">
          {t("dashboard.storage",)}
        </TabsTrigger>
      </TabsList>

      <TabsContent value="general">
        <div className="flex flex-col gap-6 pt-4">
          <ServerConfigForm />
        </div>
      </TabsContent>

      <TabsContent value="dht">
        <div className="flex flex-col gap-6 pt-4">
          <DHTConfigForm />
        </div>
      </TabsContent>

      <TabsContent value="classifier">
        <div className="flex flex-col gap-6 pt-4">
          <ClassifierLLMConfigForm />
        </div>
      </TabsContent>

      <TabsContent value="storage">
        <div className="flex flex-col gap-6 pt-4">
          <StorageConfigForm />
        </div>
      </TabsContent>
    </Tabs>
  );
}
