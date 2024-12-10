export namespace peer {
	
	export class ExpandingRing {
	    Initial: number;
	    Factor: number;
	    Retry: number;
	    Timeout: number;
	
	    static createFrom(source: any = {}) {
	        return new ExpandingRing(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Initial = source["Initial"];
	        this.Factor = source["Factor"];
	        this.Retry = source["Retry"];
	        this.Timeout = source["Timeout"];
	    }
	}

}

export namespace regexp {
	
	export class Regexp {
	
	
	    static createFrom(source: any = {}) {
	        return new Regexp(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

export namespace sync {
	
	export class WaitGroup {
	
	
	    static createFrom(source: any = {}) {
	        return new WaitGroup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

export namespace transport {
	
	export class Header {
	    PacketID: string;
	    Timestamp: number;
	    Source: string;
	    RelayedBy: string;
	    Destination: string;
	
	    static createFrom(source: any = {}) {
	        return new Header(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.PacketID = source["PacketID"];
	        this.Timestamp = source["Timestamp"];
	        this.Source = source["Source"];
	        this.RelayedBy = source["RelayedBy"];
	        this.Destination = source["Destination"];
	    }
	}
	export class Message {
	    Type: string;
	    Payload: number[];
	
	    static createFrom(source: any = {}) {
	        return new Message(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Type = source["Type"];
	        this.Payload = source["Payload"];
	    }
	}
	export class Packet {
	    Header?: Header;
	    Msg?: Message;
	
	    static createFrom(source: any = {}) {
	        return new Packet(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Header = this.convertValues(source["Header"], Header);
	        this.Msg = this.convertValues(source["Msg"], Message);
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

}

export namespace types {
	
	export class PaxosValue {
	    Filename: string;
	    Metahash: string;
	
	    static createFrom(source: any = {}) {
	        return new PaxosValue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Filename = source["Filename"];
	        this.Metahash = source["Metahash"];
	    }
	}
	export class BlockchainBlock {
	    Index: number;
	    Hash: number[];
	    Value: PaxosValue;
	    PrevHash: number[];
	
	    static createFrom(source: any = {}) {
	        return new BlockchainBlock(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Index = source["Index"];
	        this.Hash = source["Hash"];
	        this.Value = this.convertValues(source["Value"], PaxosValue);
	        this.PrevHash = source["PrevHash"];
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
	export class CRDTOperation {
	    Type: string;
	    BlockType: string;
	    Origin: string;
	    OperationId: number;
	    DocumentId: string;
	    BlockId: string;
	    Operation: any;
	
	    static createFrom(source: any = {}) {
	        return new CRDTOperation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Type = source["Type"];
	        this.BlockType = source["BlockType"];
	        this.Origin = source["Origin"];
	        this.OperationId = source["OperationId"];
	        this.DocumentId = source["DocumentId"];
	        this.BlockId = source["BlockId"];
	        this.Operation = source["Operation"];
	    }
	}
	export class CRDTOperationsMessage {
	    Operations: CRDTOperation[];
	
	    static createFrom(source: any = {}) {
	        return new CRDTOperationsMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Operations = this.convertValues(source["Operations"], CRDTOperation);
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
	export class FileInfo {
	    Name: string;
	    Metahash: string;
	    Chunks: number[][];
	
	    static createFrom(source: any = {}) {
	        return new FileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.Metahash = source["Metahash"];
	        this.Chunks = source["Chunks"];
	    }
	}
	
	export class Rumor {
	    Origin: string;
	    Sequence: number;
	    Msg?: transport.Message;
	
	    static createFrom(source: any = {}) {
	        return new Rumor(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Origin = source["Origin"];
	        this.Sequence = source["Sequence"];
	        this.Msg = this.convertValues(source["Msg"], transport.Message);
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
	export class SearchRequestMessage {
	    RequestID: string;
	    Origin: string;
	    Pattern: string;
	    Budget: number;
	
	    static createFrom(source: any = {}) {
	        return new SearchRequestMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RequestID = source["RequestID"];
	        this.Origin = source["Origin"];
	        this.Pattern = source["Pattern"];
	        this.Budget = source["Budget"];
	    }
	}

}

