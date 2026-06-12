import { ref, computed } from 'vue';

import { useServerAdminStore } from '@/stores/serverAdmin.ts';

import { AdminUser } from '@/models/server_admin.ts';

export function useServerAdminUserListPageBase() {
    const serverAdminStore = useServerAdminStore();

    const loading = ref<boolean>(true);
    const loadingError = ref<unknown>(null);
    const updating = ref<boolean>(false);

    const allUsers = computed<AdminUser[]>(() => serverAdminStore.allUsers);

    function canToggleAdmin(user: AdminUser): boolean {
        return !user.isEnvAdmin;
    }

    function reload(): Promise<AdminUser[]> {
        return serverAdminStore.loadAllUsers();
    }

    return {
        // dependent stores
        serverAdminStore,
        // states
        loading,
        loadingError,
        updating,
        // computed states
        allUsers,
        // functions
        canToggleAdmin,
        reload
    };
}
