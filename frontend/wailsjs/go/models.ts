export namespace models {
	
	export class PrintJob {
	    id: string;
	    fileName: string;
	    hospitalNo: string;
	    userName: string;
	    printer: string;
	    status: string;
	    osJobId?: string;
	    error?: string;
	    createdAt: string;
	    filePath?: string;
	
	    static createFrom(source: any = {}) {
	        return new PrintJob(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.fileName = source["fileName"];
	        this.hospitalNo = source["hospitalNo"];
	        this.userName = source["userName"];
	        this.printer = source["printer"];
	        this.status = source["status"];
	        this.osJobId = source["osJobId"];
	        this.error = source["error"];
	        this.createdAt = source["createdAt"];
	        this.filePath = source["filePath"];
	    }
	}

}

