export namespace domain {
	
	export class ConnectionInput {
	    id?: string;
	    name: string;
	    protocol: string;
	    host?: string;
	    port?: number;
	    baseUrl?: string;
	    share?: string;
	    rootPath?: string;
	    username?: string;
	    domain?: string;
	    passiveMode?: boolean;
	    password?: string;
	    savePassword?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.protocol = source["protocol"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.baseUrl = source["baseUrl"];
	        this.share = source["share"];
	        this.rootPath = source["rootPath"];
	        this.username = source["username"];
	        this.domain = source["domain"];
	        this.passiveMode = source["passiveMode"];
	        this.password = source["password"];
	        this.savePassword = source["savePassword"];
	    }
	}
	export class ConnectionProfile {
	    id: string;
	    name: string;
	    protocol: string;
	    host?: string;
	    port?: number;
	    baseUrl?: string;
	    share?: string;
	    rootPath?: string;
	    username?: string;
	    domain?: string;
	    passiveMode?: boolean;
	    credentialKey?: string;
	    passwordSaved: boolean;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.protocol = source["protocol"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.baseUrl = source["baseUrl"];
	        this.share = source["share"];
	        this.rootPath = source["rootPath"];
	        this.username = source["username"];
	        this.domain = source["domain"];
	        this.passiveMode = source["passiveMode"];
	        this.credentialKey = source["credentialKey"];
	        this.passwordSaved = source["passwordSaved"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class RemoteEntry {
	    name: string;
	    path: string;
	    type: string;
	    size: number;
	    modifiedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new RemoteEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.type = source["type"];
	        this.size = source["size"];
	        this.modifiedAt = source["modifiedAt"];
	    }
	}
	export class TransferTask {
	    id: string;
	    connectionId: string;
	    direction: string;
	    source: string;
	    destination: string;
	    status: string;
	    bytesDone: number;
	    bytesTotal: number;
	    errorMessage?: string;
	    createdAt: string;
	    startedAt?: string;
	    finishedAt?: string;
	
	    static createFrom(source: any = {}) {
	        return new TransferTask(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.connectionId = source["connectionId"];
	        this.direction = source["direction"];
	        this.source = source["source"];
	        this.destination = source["destination"];
	        this.status = source["status"];
	        this.bytesDone = source["bytesDone"];
	        this.bytesTotal = source["bytesTotal"];
	        this.errorMessage = source["errorMessage"];
	        this.createdAt = source["createdAt"];
	        this.startedAt = source["startedAt"];
	        this.finishedAt = source["finishedAt"];
	    }
	}

}

