<template>
    <v-row>
        <v-col cols="12">
            <v-card>
                <template #title>
                    <div class="title-and-toolbar d-flex align-center">
                        <span>{{ tt('User Management') }}</span>
                        <v-btn density="compact" color="default" variant="text" size="24"
                               class="ms-2" :icon="true" :disabled="loading || updating"
                               :loading="loading" @click="reloadUsers(true)">
                            <template #loader>
                                <v-progress-circular indeterminate size="20"/>
                            </template>
                            <v-icon :icon="mdiRefresh" size="24" />
                            <v-tooltip activator="parent">{{ tt('Refresh') }}</v-tooltip>
                        </v-btn>
                    </div>
                </template>

                <v-card-text>
                    <v-table class="server-admin-users-table" :hover="!loading">
                        <thead>
                        <tr>
                            <th class="text-uppercase">{{ tt('Username') }}</th>
                            <th class="text-uppercase">{{ tt('Nickname') }}</th>
                            <th class="text-uppercase" style="width: 200px">{{ tt('Administrator') }}</th>
                        </tr>
                        </thead>
                        <tbody v-if="loading && allUsers.length < 1">
                        <tr :key="itemIdx" v-for="itemIdx in [ 1, 2 ]">
                            <td class="px-0" colspan="3">
                                <v-skeleton-loader type="text" :loading="true"></v-skeleton-loader>
                            </td>
                        </tr>
                        </tbody>
                        <tbody v-else>
                        <tr :key="user.uid" v-for="user in allUsers">
                            <td>{{ user.username }}</td>
                            <td>{{ user.nickname }}</td>
                            <td>
                                <div class="d-flex align-center">
                                    <v-switch density="compact" color="primary" hide-details
                                              :disabled="loading || updating || !canToggleAdmin(user)"
                                              :model-value="user.isAdmin"
                                              @click.prevent="toggleAdmin(user)"></v-switch>
                                    <v-chip class="ms-2" size="small" v-if="user.isEnvAdmin">
                                        {{ tt('Server Settings') }}
                                        <v-tooltip activator="parent">{{ tt('This user is configured as administrator in server settings and cannot be revoked') }}</v-tooltip>
                                    </v-chip>
                                </div>
                            </td>
                        </tr>
                        </tbody>
                    </v-table>
                </v-card-text>
            </v-card>
        </v-col>
    </v-row>

    <confirm-dialog ref="confirmDialog"/>
    <snack-bar ref="snackbar" />
</template>

<script setup lang="ts">
import ConfirmDialog from '@/components/desktop/ConfirmDialog.vue';
import SnackBar from '@/components/desktop/SnackBar.vue';

import { useTemplateRef, onMounted } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useServerAdminUserListPageBase } from '@/views/base/admin/ServerAdminUserListPageBase.ts';

import type { AdminUser } from '@/models/server_admin.ts';

import {
    mdiRefresh
} from '@mdi/js';

type ConfirmDialogType = InstanceType<typeof ConfirmDialog>;
type SnackBarType = InstanceType<typeof SnackBar>;

const { tt } = useI18n();

const {
    serverAdminStore,
    loading,
    updating,
    allUsers,
    canToggleAdmin,
    reload
} = useServerAdminUserListPageBase();

const confirmDialog = useTemplateRef<ConfirmDialogType>('confirmDialog');
const snackbar = useTemplateRef<SnackBarType>('snackbar');

function reloadUsers(force: boolean): void {
    loading.value = true;

    reload().then(() => {
        loading.value = false;

        if (force) {
            snackbar.value?.showMessage('User list has been updated');
        }
    }).catch(error => {
        loading.value = false;

        if (!error.processed) {
            snackbar.value?.showError(error);
        }
    });
}

function toggleAdmin(user: AdminUser): void {
    if (loading.value || updating.value || !canToggleAdmin(user)) {
        return;
    }

    const confirmMessage = user.isAdmin
        ? 'Are you sure you want to revoke administrator privileges from this user?'
        : 'Are you sure you want to grant administrator privileges to this user?';

    confirmDialog.value?.open(confirmMessage).then(() => {
        updating.value = true;

        const promise = user.isAdmin
            ? serverAdminStore.revokeAdmin({ uid: user.uid })
            : serverAdminStore.grantAdmin({ uid: user.uid });

        promise.then(() => {
            updating.value = false;
        }).catch(error => {
            updating.value = false;

            if (!error.processed) {
                snackbar.value?.showError(error);
            }
        });
    });
}

onMounted(() => {
    serverAdminStore.loadPermission({}).then(administrable => {
        if (administrable) {
            reloadUsers(false);
        } else {
            loading.value = false;
        }
    });
});
</script>
