<template>
    <div>
        <v-tabs show-arrows v-model="activeTab">
            <v-tab value="basicSetting" @click="pushRouter('basicSetting')">
                <v-icon size="20" start :icon="mdiCogOutline"/>
                {{ tt('Basic') }}
            </v-tab>
            <v-tab value="applicationLockSetting" @click="pushRouter('applicationLockSetting')">
                <v-icon size="20" start :icon="mdiLockOpenOutline"/>
                {{ tt('Application Lock') }}
            </v-tab>
            <v-tab value="statisticsSetting" @click="pushRouter('statisticsSetting')">
                <v-icon size="20" start :icon="mdiChartPieOutline"/>
                {{ tt('Statistics') }}
            </v-tab>
            <v-tab value="cloudSyncSetting" @click="pushRouter('cloudSyncSetting')">
                <v-icon size="20" start :icon="mdiCloudOutline"/>
                {{ tt('Settings Sync') }}
            </v-tab>
            <v-tab value="browserCacheSetting" @click="pushRouter('browserCacheSetting')">
                <v-icon size="20" start :icon="mdiDatabaseClockOutline"/>
                {{ tt('Browser Cache Management') }}
            </v-tab>
            <v-tab value="llmPromptSetting" @click="pushRouter('llmPromptSetting')" v-if="isTransactionFromAIImageRecognitionEnabled()">
                <v-icon size="20" start :icon="mdiTextBoxSearchOutline"/>
                {{ tt('AI Receipt Recognition Prompts') }}
            </v-tab>
            <v-tab value="serverAdminSetting" @click="pushRouter('serverAdminSetting')" v-if="serverAdminStore.administrable">
                <v-icon size="20" start :icon="mdiAccountCogOutline"/>
                {{ tt('Server Administration') }}
            </v-tab>
        </v-tabs>

        <v-window class="mt-4 disable-tab-transition" v-model="activeTab">
            <v-window-item value="basicSetting">
                <app-basic-setting-tab/>
            </v-window-item>

            <v-window-item value="applicationLockSetting">
                <app-lock-setting-tab/>
            </v-window-item>

            <v-window-item value="statisticsSetting">
                <app-statistics-setting-tab/>
            </v-window-item>

            <v-window-item value="cloudSyncSetting">
                <app-cloud-sync-setting-tab/>
            </v-window-item>

            <v-window-item value="browserCacheSetting">
                <app-browser-cache-setting-tab/>
            </v-window-item>

            <v-window-item value="llmPromptSetting" v-if="isTransactionFromAIImageRecognitionEnabled()">
                <app-llm-prompt-setting-tab/>
            </v-window-item>

            <v-window-item value="serverAdminSetting" v-if="serverAdminStore.administrable">
                <app-server-admin-setting-tab/>
            </v-window-item>
        </v-window>
    </div>
</template>

<script setup lang="ts">
import AppBasicSettingTab from './settings/tabs/AppBasicSettingTab.vue';
import AppLockSettingTab from './settings/tabs/AppLockSettingTab.vue';
import AppStatisticsSettingTab from './settings/tabs/AppStatisticsSettingTab.vue';
import AppCloudSyncSettingTab from './settings/tabs/AppCloudSyncSettingTab.vue';
import AppBrowserCacheSettingTab from './settings/tabs/AppBrowserCacheSettingTab.vue';
import AppLlmPromptSettingTab from './settings/tabs/AppLlmPromptSettingTab.vue';
import AppServerAdminSettingTab from './settings/tabs/AppServerAdminSettingTab.vue';

import { ref } from 'vue';
import { useRouter, onBeforeRouteUpdate } from 'vue-router';

import { useI18n } from '@/locales/helpers.ts';

import { useServerAdminStore } from '@/stores/serverAdmin.ts';

import { isTransactionFromAIImageRecognitionEnabled } from '@/lib/server_settings.ts';

import {
    mdiCogOutline,
    mdiLockOpenOutline,
    mdiChartPieOutline,
    mdiCloudOutline,
    mdiDatabaseClockOutline,
    mdiTextBoxSearchOutline,
    mdiAccountCogOutline
} from '@mdi/js';

const props = defineProps<{
    initTab?: string;
}>();

const router = useRouter();

const { tt } = useI18n();

const ALL_TABS: string[] = [
    'basicSetting',
    'applicationLockSetting',
    'statisticsSetting',
    'cloudSyncSetting',
    'browserCacheSetting'
];

if (isTransactionFromAIImageRecognitionEnabled()) {
    ALL_TABS.push('llmPromptSetting');
}

const serverAdminStore = useServerAdminStore();
serverAdminStore.loadPermission({});
ALL_TABS.push('serverAdminSetting');

const activeTab = ref<string>((() => {
    let queryActiveTab = props.initTab || 'basicSetting';

    if (ALL_TABS.indexOf(queryActiveTab) < 0) {
        queryActiveTab = 'basicSetting';
    }

    return queryActiveTab;
})());

const pushRouter = (tab: string) => {
    router.push(`/app/settings?tab=${tab}`);
};

onBeforeRouteUpdate((to) => {
    if (to.query && to.query['tab'] && ALL_TABS.indexOf(to.query['tab'] as string) >= 0) {
        activeTab.value = to.query['tab'] as string;
    } else {
        activeTab.value = 'basicSetting';
    }
});
</script>
