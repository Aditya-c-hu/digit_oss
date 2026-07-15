/* tslint:disable */
/* eslint-disable */
// Generated using typescript-generator version 2.22.595 on 2026-05-21 15:37:41.

export namespace Digit {

    interface BoundaryRequest {
        RequestInfo: RequestInfo;
        Boundary: BoundaryLocation;
    }

    interface BoundaryResponse {
        ResponseInfo: ResponseInfo;
        Boundary: Boundary[];
    }

    interface BoundaryMdmsResponse {
        ResponseInfo: ResponseInfo;
        TenantBoundary: MdmsTenantBoundary[];
    }

    interface BoundarySearchRequest {
        RequestInfo: RequestInfo;
        Boundary: Boundary;
    }

    interface BoundaryTypeRequest {
        RequestInfo: RequestInfo;
        BoundaryType: BoundaryType;
    }

    interface BoundaryTypeResponse {
        ResponseInfo: ResponseInfo;
        BoundaryType: BoundaryType[];
    }

    interface BoundaryTypeSearchRequest {
        RequestInfo: RequestInfo;
        BoundaryType: BoundaryLocationType;
    }

    interface CityRequest {
        RequestInfo: RequestInfo;
        City: City;
    }

    interface CityResponse {
        ResponseInfo: ResponseInfo;
        City: City;
    }

    interface CrossHierarchyRequest {
        RequestInfo: RequestInfo;
        CrossHierarchy: CrossHierarchy;
    }

    interface CrossHierarchyResponse {
        ResponseInfo: ResponseInfo;
        CrossHierarchy: CrossHierarchy[];
    }

    interface CrossHierarchySearchRequest {
        RequestInfo: RequestInfo;
        CrossHierarchy: CrossHierarchySearchModel;
    }

    interface GeographicalResponse {
        ResponseInfo: ResponseInfo;
        Geography: Geography[];
    }

    interface HierarchyTypeRequest {
        RequestInfo: RequestInfo;
        HierarchyType: HierarchyType;
    }

    interface HierarchyTypeResponse {
        ResponseInfo: ResponseInfo;
        HierarchyType: HierarchyType[];
    }

    interface HierarchyTypeSearchRequest {
        RequestInfo: RequestInfo;
        HierarchyType: HierarchyTypeSearchModel;
    }

    interface MdmsRequest {
        RequestInfo: RequestInfo;
        MdmsCriteria: MdmsCriteria;
    }

    interface MdmsResponse {
        ResponseInfo: ResponseInfo;
        MdmsRes: { [index: string]: { [index: string]: any[] } };
    }

    interface ShapeFileResponse {
        ResponseInfo: ResponseInfo;
        ShapeFile: ShapeFile;
    }

    interface TenantResponse {
        ResponseInfo: ResponseInfo;
        Tenant: Tenant;
    }

    interface BoundaryLocation {
        id: number;
        name: string;
        boundaryNum: number;
        code: string;
        area: string;
        codes: string;
        boundaryType: BoundaryType;
        parent: BoundaryLocation;
        fromDate: Date;
        toDate: Date;
        bndryId: number;
        localName: string;
        longitude: number;
        latitude: number;
        materializedPath: string;
        tenantId: string;
        createdBy: number;
        createdDate: number;
        lastModifiedBy: number;
        lastModifiedDate: number;
        history: boolean;
    }

    interface Boundary {
        id: string;
        name: string;
        longitude: number;
        latitude: number;
        boundaryNum: number;
        parent: Boundary;
        tenantId: string;
        boundaryType: BoundaryType;
        code: string;
        area: string;
    }

    interface MdmsTenantBoundary {
        hierarchyType: HierarchyType;
        boundary: MdmsBoundary[];
        tenantId: string;
    }

    interface BoundaryType {
        createdBy: number;
        createdDate: number;
        lastModifiedBy: number;
        lastModifiedDate: number;
        id: string;
        name: string;
        code: string;
        hierarchyType: HierarchyType;
        parent: BoundaryType;
        hierarchy: number;
        localName: string;
        parentName: string;
        childBoundaryTypes: BoundaryType[];
        tenantId: string;
    }

    interface BoundaryLocationType {
        id: number;
        name: string;
        code: string;
        hierarchyType: HierarchyType;
        parent: BoundaryLocationType;
        hierarchy: number;
        localName: string;
        parentName: string;
        childBoundaryTypes: BoundaryLocationType[];
        tenantId: string;
    }

    interface City {
        id: string;
        code: string;
        name: string;
        districtCode: string;
        districtName: string;
        grade: string;
        domainURL: string;
        regionName: string;
        tenantId: string;
    }

    interface CrossHierarchy {
        id: number;
        parent: BoundaryLocation;
        code: string;
        child: BoundaryLocation;
        tenantId: string;
        createdBy: number;
        createdDate: Date;
        lastModifiedBy: number;
        lastModifiedDate: Date;
    }

    interface CrossHierarchySearchModel {
        id: number;
        code: string;
        tenantId: string;
    }

    interface Geography {
        name: string;
        division: string;
        geoJson: any;
        geoJsonChildren: any;
        tenantId: string;
    }

    interface HierarchyType {
        id: number;
        name: string;
        code: string;
        localName: string;
        tenantId: string;
        createdBy: number;
        createdDate: number;
        lastModifiedBy: number;
        lastModifiedDate: number;
        version: number;
        new: boolean;
    }

    interface HierarchyTypeSearchModel {
        id: number;
        name: string;
        code: string;
        localName: string;
        tenantId: string;
    }

    interface MdmsCriteria {
        tenantId: string;
        moduleDetails: ModuleDetails[];
    }

    interface ShapeFile {
        fileExist: boolean;
    }

    interface Tenant {
        id: number;
        code: string;
        name: string;
        description: string;
        logoId: string;
        imageId: string;
        domainUrl: string;
        type: TenantType;
        twitterUrl: string;
        facebookUrl: string;
        emailId: string;
        address: string;
        contactNumber: string;
        helpLineNumber: string;
        city: TenantCity;
        contactNumberInvalidLength: boolean;
        codeOfInvalidLength: boolean;
        helpLineNumberInvalidLength: boolean;
        cityAbsent: boolean;
        codeAbsent: boolean;
        nameAbsent: boolean;
        imageIdAbsent: boolean;
        typeAbsent: boolean;
        typeInvalid: boolean;
        logoIdAbsent: boolean;
    }

    interface MdmsBoundary {
        code: string;
        name: string;
        label: string;
        latitude: string;
        longitude: string;
        area: string;
        pincode: number[];
        boundaryNum: number;
        children: MdmsBoundary[];
    }

    interface ModuleDetails {
        masterDetails: MasterDetails[];
        moduleName: string;
    }

    interface TenantCity {
        id: number;
        name: string;
        localName: string;
        districtCode: string;
        districtName: string;
        regionName: string;
        longitude: number;
        latitude: number;
        tenantCode: string;
        ulbGrade: string;
        createdBy: number;
        createdDate: Date;
        lastModifiedBy: number;
        lastModifiedDate: Date;
        shapeFileLocation: string;
        captcha: string;
        code: string;
        valid: boolean;
        nameAbsent: boolean;
        ulbgradeAbsent: boolean;
    }

    interface MasterDetails {
        name: string;
        filter: string;
    }

    type TenantType = "CITY";

}
