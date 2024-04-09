// source: cosmos/msg/v1/msg.proto
/**
 * @fileoverview
 * @enhanceable
 * @suppress {missingRequire} reports error on implicit type usages.
 * @suppress {messageConventions} JS Compiler reports an error if a variable or
 *     field starts with 'MSG_' and isn't a translatable message.
 * @public
 */
// GENERATED CODE -- DO NOT EDIT!
/* eslint-disable */
// @ts-nocheck

var jspb = require('google-protobuf');
var goog = jspb;
var global =
    (typeof globalThis !== 'undefined' && globalThis) ||
    (typeof window !== 'undefined' && window) ||
    (typeof global !== 'undefined' && global) ||
    (typeof self !== 'undefined' && self) ||
    (function () { return this; }).call(null) ||
    Function('return this')();

var google_protobuf_descriptor_pb = require('google-protobuf/google/protobuf/descriptor_pb.js');
goog.object.extend(proto, google_protobuf_descriptor_pb);
goog.exportSymbol('proto.cosmos.msg.v1.service', null, global);
goog.exportSymbol('proto.cosmos.msg.v1.signerList', null, global);

/**
 * A tuple of {field number, class constructor} for the extension
 * field named `service`.
 * @type {!jspb.ExtensionFieldInfo<boolean>}
 */
proto.cosmos.msg.v1.service = new jspb.ExtensionFieldInfo(
    11110000,
    {service: 0},
    null,
     /** @type {?function((boolean|undefined),!jspb.Message=): !Object} */ (
         null),
    0);

google_protobuf_descriptor_pb.ServiceOptions.extensionsBinary[11110000] = new jspb.ExtensionFieldBinaryInfo(
    proto.cosmos.msg.v1.service,
    jspb.BinaryReader.prototype.readBool,
    jspb.BinaryWriter.prototype.writeBool,
    undefined,
    undefined,
    false);
// This registers the extension field with the extended class, so that
// toObject() will function correctly.
google_protobuf_descriptor_pb.ServiceOptions.extensions[11110000] = proto.cosmos.msg.v1.service;


/**
 * A tuple of {field number, class constructor} for the extension
 * field named `signerList`.
 * @type {!jspb.ExtensionFieldInfo<!Array<string>>}
 */
proto.cosmos.msg.v1.signerList = new jspb.ExtensionFieldInfo(
    11110000,
    {signerList: 0},
    null,
     /** @type {?function((boolean|undefined),!jspb.Message=): !Object} */ (
         null),
    1);

google_protobuf_descriptor_pb.MessageOptions.extensionsBinary[11110000] = new jspb.ExtensionFieldBinaryInfo(
    proto.cosmos.msg.v1.signerList,
    jspb.BinaryReader.prototype.readString,
    jspb.BinaryWriter.prototype.writeRepeatedString,
    undefined,
    undefined,
    false);
// This registers the extension field with the extended class, so that
// toObject() will function correctly.
google_protobuf_descriptor_pb.MessageOptions.extensions[11110000] = proto.cosmos.msg.v1.signerList;

goog.object.extend(exports, proto.cosmos.msg.v1);
