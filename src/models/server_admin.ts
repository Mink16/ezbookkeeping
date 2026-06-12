export class AdminUser {
    public uid: string;
    public username: string;
    public nickname: string;
    public isAdmin: boolean;
    public isEnvAdmin: boolean;

    private constructor(uid: string, username: string, nickname: string, isAdmin: boolean, isEnvAdmin: boolean) {
        this.uid = uid;
        this.username = username;
        this.nickname = nickname;
        this.isAdmin = isAdmin;
        this.isEnvAdmin = isEnvAdmin;
    }

    public static of(userResponse: AdminUserInfoResponse): AdminUser {
        return new AdminUser(userResponse.uid, userResponse.username, userResponse.nickname, userResponse.isAdmin, userResponse.isEnvAdmin);
    }

    public static ofMulti(userResponses: AdminUserInfoResponse[]): AdminUser[] {
        const users: AdminUser[] = [];

        for (const userResponse of userResponses) {
            users.push(AdminUser.of(userResponse));
        }

        return users;
    }
}

export interface AdminStatusResponse {
    readonly isAdmin: boolean;
}

export interface AdminUserInfoResponse {
    readonly uid: string;
    readonly username: string;
    readonly nickname: string;
    readonly isAdmin: boolean;
    readonly isEnvAdmin: boolean;
}

export interface AdminGrantRequest {
    readonly uid: string;
}

export interface AdminRevokeRequest {
    readonly uid: string;
}
