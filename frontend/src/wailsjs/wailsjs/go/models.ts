export namespace main {
	
	export class TokenInfo {
	    token: string;
	    userId: string;
	    active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TokenInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.token = source["token"];
	        this.userId = source["userId"];
	        this.active = source["active"];
	    }
	}
	export class apiResp__mingot_internal_profile_ArrangeResult_ {
	    success: boolean;
	    data?: profile.ArrangeResult;
	    error?: string;
	    code?: string;
	    profileId?: string;
	
	    static createFrom(source: any = {}) {
	        return new apiResp__mingot_internal_profile_ArrangeResult_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], profile.ArrangeResult);
	        this.error = source["error"];
	        this.code = source["code"];
	        this.profileId = source["profileId"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class apiResp__mingot_internal_profile_CreateResult_ {
	    success: boolean;
	    data?: profile.CreateResult;
	    error?: string;
	    code?: string;
	    profileId?: string;
	
	    static createFrom(source: any = {}) {
	        return new apiResp__mingot_internal_profile_CreateResult_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], profile.CreateResult);
	        this.error = source["error"];
	        this.code = source["code"];
	        this.profileId = source["profileId"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class apiResp__mingot_internal_profile_ImportResult_ {
	    success: boolean;
	    data?: profile.ImportResult;
	    error?: string;
	    code?: string;
	    profileId?: string;
	
	    static createFrom(source: any = {}) {
	        return new apiResp__mingot_internal_profile_ImportResult_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], profile.ImportResult);
	        this.error = source["error"];
	        this.code = source["code"];
	        this.profileId = source["profileId"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class apiResp__mingot_internal_profile_StartResult_ {
	    success: boolean;
	    data?: profile.StartResult;
	    error?: string;
	    code?: string;
	    profileId?: string;
	
	    static createFrom(source: any = {}) {
	        return new apiResp__mingot_internal_profile_StartResult_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], profile.StartResult);
	        this.error = source["error"];
	        this.code = source["code"];
	        this.profileId = source["profileId"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class apiResp___main_TokenInfo_ {
	    success: boolean;
	    data?: TokenInfo[];
	    error?: string;
	    code?: string;
	    profileId?: string;
	
	    static createFrom(source: any = {}) {
	        return new apiResp___main_TokenInfo_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], TokenInfo);
	        this.error = source["error"];
	        this.code = source["code"];
	        this.profileId = source["profileId"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class apiResp___mingot_internal_profile_Summary_ {
	    success: boolean;
	    data?: profile.Summary[];
	    error?: string;
	    code?: string;
	    profileId?: string;
	
	    static createFrom(source: any = {}) {
	        return new apiResp___mingot_internal_profile_Summary_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], profile.Summary);
	        this.error = source["error"];
	        this.code = source["code"];
	        this.profileId = source["profileId"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class apiResp___uint8_ {
	    success: boolean;
	    data?: number[];
	    error?: string;
	    code?: string;
	    profileId?: string;
	
	    static createFrom(source: any = {}) {
	        return new apiResp___uint8_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = source["data"];
	        this.error = source["error"];
	        this.code = source["code"];
	        this.profileId = source["profileId"];
	    }
	}
	export class apiResp_interface____ {
	    success: boolean;
	    data?: any;
	    error?: string;
	    code?: string;
	    profileId?: string;
	
	    static createFrom(source: any = {}) {
	        return new apiResp_interface____(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = source["data"];
	        this.error = source["error"];
	        this.code = source["code"];
	        this.profileId = source["profileId"];
	    }
	}

}

export namespace profile {
	
	export class ArrangeResult {
	    arrangedCount: number;
	
	    static createFrom(source: any = {}) {
	        return new ArrangeResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.arrangedCount = source["arrangedCount"];
	    }
	}
	export class CreateResult {
	    id: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	    }
	}
	export class ImportResult {
	    id: string;
	
	    static createFrom(source: any = {}) {
	        return new ImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	    }
	}
	export class StartResult {
	    port: number;
	
	    static createFrom(source: any = {}) {
	        return new StartResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.port = source["port"];
	    }
	}
	export class Summary {
	    id: string;
	    name: string;
	    port?: number;
	    status: string;
	    folder: string;
	
	    static createFrom(source: any = {}) {
	        return new Summary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.port = source["port"];
	        this.status = source["status"];
	        this.folder = source["folder"];
	    }
	}

}

