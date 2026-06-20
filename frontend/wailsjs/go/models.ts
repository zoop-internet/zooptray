export namespace main {
	
	export class ApiConfig {
	    endpoint: string;
	    heartbeatIntervalSeconds: number;
	
	    static createFrom(source: any = {}) {
	        return new ApiConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.endpoint = source["endpoint"];
	        this.heartbeatIntervalSeconds = source["heartbeatIntervalSeconds"];
	    }
	}
	export class SystemConfig {
	    autoStartOnBoot: boolean;
	    preferredInterface: string;
	
	    static createFrom(source: any = {}) {
	        return new SystemConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.autoStartOnBoot = source["autoStartOnBoot"];
	        this.preferredInterface = source["preferredInterface"];
	    }
	}
	export class SharingConfig {
	    enabled: boolean;
	    maxClients: number;
	    bandwidthCapPercent: number;
	    routingPolicy: string;
	
	    static createFrom(source: any = {}) {
	        return new SharingConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.maxClients = source["maxClients"];
	        this.bandwidthCapPercent = source["bandwidthCapPercent"];
	        this.routingPolicy = source["routingPolicy"];
	    }
	}
	export class NodeConfig {
	    id: string;
	    displayName: string;
	    privateKeyPath: string;
	    isPublic: boolean;
	
	    static createFrom(source: any = {}) {
	        return new NodeConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.displayName = source["displayName"];
	        this.privateKeyPath = source["privateKeyPath"];
	        this.isPublic = source["isPublic"];
	    }
	}
	export class Config {
	    node: NodeConfig;
	    sharing: SharingConfig;
	    system: SystemConfig;
	    api: ApiConfig;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.node = this.convertValues(source["node"], NodeConfig);
	        this.sharing = this.convertValues(source["sharing"], SharingConfig);
	        this.system = this.convertValues(source["system"], SystemConfig);
	        this.api = this.convertValues(source["api"], ApiConfig);
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
	export class LogMessage {
	    timestamp: string;
	    level: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new LogMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timestamp = source["timestamp"];
	        this.level = source["level"];
	        this.message = source["message"];
	    }
	}
	
	

}

