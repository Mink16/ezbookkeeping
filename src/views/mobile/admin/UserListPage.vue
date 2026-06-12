<template>
    <f7-page ptr @ptr:refresh="reload">
        <f7-navbar>
            <f7-nav-left :class="{ 'disabled': loading }" :back-link="tt('Back')"></f7-nav-left>
            <f7-nav-title :title="tt('User Management')"></f7-nav-title>
        </f7-navbar>

        <f7-list strong inset dividers class="margin-top skeleton-text" v-if="loading">
            <f7-list-item title="Username" footer="Nickname"
                          :key="itemIdx" v-for="itemIdx in [ 1, 2 ]">
                <template #media>
                    <f7-icon f7="person"></f7-icon>
                </template>
            </f7-list-item>
        </f7-list>

        <f7-list strong inset dividers class="margin-top server-admin-user-list" v-else-if="!loading">
            <f7-list-item :title="user.username"
                          :footer="getUserFooter(user)"
                          :key="user.uid"
                          v-for="user in allUsers">
                <template #media>
                    <f7-icon :f7="user.isAdmin ? 'person_badge_plus' : 'person'"></f7-icon>
                </template>
                <template #after>
                    <f7-toggle :checked="user.isAdmin"
                               :disabled="loading || updating || !canToggleAdmin(user)"
                               @toggle:change="toggleAdmin(user, $event)"></f7-toggle>
                </template>
            </f7-list-item>
        </f7-list>
    </f7-page>
</template>

<script setup lang="ts">
import type { Router } from 'framework7/types';

import { useI18n } from '@/locales/helpers.ts';
import { useI18nUIComponents, showLoading, hideLoading } from '@/lib/ui/mobile.ts';

import { useServerAdminUserListPageBase } from '@/views/base/admin/ServerAdminUserListPageBase.ts';

import type { AdminUser } from '@/models/server_admin.ts';

const props = defineProps<{
    f7router: Router.Router;
}>();

const { tt } = useI18n();
const { showConfirm, showToast, routeBackOnError } = useI18nUIComponents();

const {
    serverAdminStore,
    loading,
    loadingError,
    updating,
    allUsers,
    canToggleAdmin,
    reload: reloadUserList
} = useServerAdminUserListPageBase();

function getUserFooter(user: AdminUser): string | undefined {
    if (user.isEnvAdmin) {
        return (user.nickname ? user.nickname + ' · ' : '') + tt('This user is configured as administrator in server settings and cannot be revoked');
    }

    return user.nickname || undefined;
}

function init(): void {
    loading.value = true;

    serverAdminStore.loadPermission({}).then(administrable => {
        if (!administrable) {
            props.f7router.back();
            return;
        }

        reloadUserList().then(() => {
            loading.value = false;
        }).catch(error => {
            if (error.processed) {
                loading.value = false;
            } else {
                loadingError.value = error;
                showToast(error.message || error);
            }
        });
    });
}

function reload(done?: () => void): void {
    reloadUserList().then(() => {
        done?.();

        if (done) {
            showToast('User list has been updated');
        }
    }).catch(error => {
        done?.();

        if (!error.processed) {
            showToast(error.message || error);
        }
    });
}

function toggleAdmin(user: AdminUser, newValue: boolean): void {
    if (loading.value || updating.value || !canToggleAdmin(user) || newValue === user.isAdmin) {
        return;
    }

    const confirmMessage = user.isAdmin
        ? 'Are you sure you want to revoke administrator privileges from this user?'
        : 'Are you sure you want to grant administrator privileges to this user?';

    showConfirm(confirmMessage, () => {
        updating.value = true;
        showLoading();

        const promise = user.isAdmin
            ? serverAdminStore.revokeAdmin({ uid: user.uid })
            : serverAdminStore.grantAdmin({ uid: user.uid });

        promise.then(() => {
            updating.value = false;
            hideLoading();
        }).catch(error => {
            updating.value = false;
            hideLoading();

            if (!error.processed) {
                showToast(error.message || error);
            }
        });
    }, () => {
        // user cancelled, restore the toggle state from the store value
        reloadUserList().catch(() => { /* ignored */ });
    });
}

routeBackOnError(props.f7router, loadingError);
init();
</script>

<style>
.server-admin-user-list {
    --f7-list-item-footer-font-size: var(--ebk-large-footer-font-size);
}
</style>
