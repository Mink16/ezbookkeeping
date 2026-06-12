import { ref } from 'vue';
import { defineStore } from 'pinia';

import { AdminUser } from '@/models/server_admin.ts';

import logger from '@/lib/logger.ts';
import services from '@/lib/services.ts';

let pendingPermissionPromise: Promise<boolean> | null = null;

export const useServerAdminStore = defineStore('serverAdmin', () => {
    const administrable = ref<boolean>(false);
    const permissionResolved = ref<boolean>(false);
    const allUsers = ref<AdminUser[]>([]);

    function resetServerAdmin(): void {
        administrable.value = false;
        permissionResolved.value = false;
        allUsers.value = [];
        pendingPermissionPromise = null;
    }

    // resolves whether the current user is a server admin, a failing request resolves to false
    // (the admin-only ui is simply hidden) and never rejects so that callers do not need error handling
    function loadPermission({ force }: { force?: boolean }): Promise<boolean> {
        if (!force && permissionResolved.value) {
            return Promise.resolve(administrable.value);
        }

        if (pendingPermissionPromise) {
            return pendingPermissionPromise;
        }

        pendingPermissionPromise = new Promise(resolve => {
            services.getServerAdminStatus().then(response => {
                const data = response.data;

                administrable.value = !!(data && data.success && data.result && data.result.isAdmin);
                permissionResolved.value = true;
                pendingPermissionPromise = null;

                resolve(administrable.value);
            }).catch(error => {
                logger.error('failed to load server admin status', error);

                administrable.value = false;
                permissionResolved.value = true;
                pendingPermissionPromise = null;

                resolve(false);
            });
        });

        return pendingPermissionPromise;
    }

    function loadAllUsers(): Promise<AdminUser[]> {
        return new Promise((resolve, reject) => {
            services.getAllServerUsers().then(response => {
                const data = response.data;

                if (!data || !data.success || !data.result) {
                    reject({ message: 'Unable to retrieve user list' });
                    return;
                }

                allUsers.value = AdminUser.ofMulti(data.result);

                resolve(allUsers.value);
            }).catch(error => {
                logger.error('failed to load user list', error);

                if (error.response && error.response.data && error.response.data.errorMessage) {
                    reject({ error: error.response.data });
                } else if (!error.processed) {
                    reject({ message: 'Unable to retrieve user list' });
                } else {
                    reject(error);
                }
            });
        });
    }

    function grantAdmin({ uid }: { uid: string }): Promise<boolean> {
        return new Promise((resolve, reject) => {
            services.grantServerAdmin({ uid }).then(response => {
                const data = response.data;

                if (!data || !data.success || !data.result) {
                    reject({ message: 'Unable to grant administrator privileges' });
                    return;
                }

                for (const user of allUsers.value) {
                    if (user.uid === uid) {
                        user.isAdmin = true;
                    }
                }

                resolve(data.result);
            }).catch(error => {
                logger.error('failed to grant admin', error);

                if (error.response && error.response.data && error.response.data.errorMessage) {
                    reject({ error: error.response.data });
                } else if (!error.processed) {
                    reject({ message: 'Unable to grant administrator privileges' });
                } else {
                    reject(error);
                }
            });
        });
    }

    function revokeAdmin({ uid }: { uid: string }): Promise<boolean> {
        return new Promise((resolve, reject) => {
            services.revokeServerAdmin({ uid }).then(response => {
                const data = response.data;

                if (!data || !data.success || !data.result) {
                    reject({ message: 'Unable to revoke administrator privileges' });
                    return;
                }

                for (const user of allUsers.value) {
                    if (user.uid === uid) {
                        user.isAdmin = false;
                    }
                }

                resolve(data.result);
            }).catch(error => {
                logger.error('failed to revoke admin', error);

                if (error.response && error.response.data && error.response.data.errorMessage) {
                    reject({ error: error.response.data });
                } else if (!error.processed) {
                    reject({ message: 'Unable to revoke administrator privileges' });
                } else {
                    reject(error);
                }
            });
        });
    }

    return {
        // states
        administrable,
        permissionResolved,
        allUsers,
        // functions
        resetServerAdmin,
        loadPermission,
        loadAllUsers,
        grantAdmin,
        revokeAdmin
    };
});
