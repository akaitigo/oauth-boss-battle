export interface BossResult {
	success: boolean;
	defeated: boolean;
	message: string;
	explanation?: string;
	code?: string;
	rfc_link?: string;
}

export interface BossInfo {
	id: number;
	name: string;
	description: string;
	defeated: boolean;
}
