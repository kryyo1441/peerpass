export namespace main {
	
	export class RoomInfo {
	    room: string;
	    peerId: string;
	    peers: string[];
	    host: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RoomInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.room = source["room"];
	        this.peerId = source["peerId"];
	        this.peers = source["peers"];
	        this.host = source["host"];
	    }
	}

}

