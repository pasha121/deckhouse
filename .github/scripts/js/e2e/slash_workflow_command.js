const {abortFailedE2eCommand, knownLabels, knownProviders} = require("../constants");
const {parseCommandArgumentAsRef, pullRequestInfo} = require("../ci");

/**
 * Build valid return object
 * @param {string} flag for realease issue logic
 * @param {object} workflow_id - workflow id
 * @param {object} targetRef - workflow target ref
 * @param {object} inputs - target workflow inputs
 * @return {object}
 */
function buildReturn(flag, workflow_id, targetRef, inputs) {
  return {
    [flag]: true,
    workflow_id,
    targetRef,
    inputs,
  }
}

/**
 * Try parse e2e abort arguments
 * @param {object} inputs
 * @param {object} inputs.core - A reference to the '@actions/core' package.
 * @param {object} inputs.github - A pre-authenticated octokit/rest.js client with pagination plugins.
 * @param {object} inputs.context - A reference to context https://github.com/actions/toolkit/blob/main/packages/github/src/context.ts#L6
 * @param {string} inputs.argv - array of slash command argv[0] is commnad
 * @return {object}
 */
function tryParseAbortE2eCluster({argv, context, core, github, ref}){
  const command = argv[0];
  if (command !== abortFailedE2eCommand) {
    return null;
  }

  // example
  // /e2e/abort static;Static;containerd;1.21 3318607912 3318607912-1-con-1-21
  // explain:
  // /e2e/abort - command
  // static;Static;containerd;1.21 - run parameters (provider;layout;cri;k8s version)
  // 3318607912 - run id (needs for get artifact)
  // 3318607912-1-con-1-21 - cluster prefix (needs for run dhctl bootstrap-phase abort command)
  // /sys/deckhouse-oss/install:pr2896 - install image path: for run necessary installer
  // user@127.0.0.1 - [additional] connection string, needs for fully bootstrapped cluster, but e2e was failed.
  //                  we  need it for destroy
  if (argv.length !== 5) {
    return {err: 'clean failed e2e cluster should have 4 arguments'};
  }

  const ranForSplit = argv[1].split(';').map(v => v.trim()).filter(v => !!v);
  if (ranForSplit.length !== 4) {
    return {err: '"ran parameters" argument should split on 4 parts'};
  }

  const run_id = argv[2];
  const cluster_prefix = argv[3];
  const installer_image_path = argv[4];
  let sshConnectStr = '';
  if (argv.length === 6) {
    sshConnectStr = argv[5] || '';
  }

  const prNumber = context.payload.issue.number;
  const pull_request_ref = ref;

  core.debug(`pull request info: ${JSON.stringify({prNumber, installer_image_path, pull_request_ref})}`);

  const provider = ranForSplit[0];
  const layout = ranForSplit[1];
  const cri = ranForSplit[2];
  const k8s_version = ranForSplit[3];
  const k8sSlug = k8s_version.replace('.', '_');
  const state_artifact_name = `failed_cluster_state_${provider}_${cri}_${k8sSlug}`;

  const inputs = {
    run_id,
    state_artifact_name,
    cluster_prefix,
    installer_image_path,
    ssh_connection_string: sshConnectStr,

    layout,
    cri,
    k8s_version,
    issue_number: prNumber.toString(),
  };

  core.debug(`e2e abort inputs: ${JSON.stringify(inputs)}`)

  return buildReturn('isDestroyFailedE2e', `e2e-clean-${provider}.yml`,'refs/heads/main', inputs)
}


/**
 * Try to parse start e2e arguments
 * @param {object} inputs
 * @param {object} inputs.core - A reference to the '@actions/core' package.
 * @param {object} inputs.github - A pre-authenticated octokit/rest.js client with pagination plugins.
 * @param {object} inputs.context - A reference to context https://github.com/actions/toolkit/blob/main/packages/github/src/context.ts#L6
 * @param {string} inputs.argv - array of slash command argv[0] is commnad
 * @param {string} inputs.ref - reference for checkout
 */
function tryParseRunE2e({argv, context, core, github, ref}){
  const command = argv[0];
  // Detect /e2e/run/* commands and /e2e/use/* arguments.
  const isE2E = Object.entries(knownLabels)
    .some(([name, info]) => {
      return info.type.startsWith('e2e') && command.startsWith('/'+name)
    })

  if(!isE2E) {
    return null;
  }

  if (argv.length === 1) {
    return {err: "Target refs is required"}
  }

  // Initial ref for e2e/run with 2 args.
  let initialRef = null
    // A ref for workflow and a target ref for e2e release update test.
  let targetRef = parseCommandArgumentAsRef(argv[1])
  if (argv.length === 3) {
    initialRef = targetRef
    targetRef = parseCommandArgumentAsRef(argv[2])
  }

  if(targetRef.err) {
    return { err: targetRef.err}
  }

  if (initialRef && initialRef.err ) {
    return { err: targetRef.err}
  }

  let workflowID = '';

  for (const provider of knownProviders) {
    if (command.includes(provider)) {
      workflowID = `e2e-${provider}.yml`;
      break;
    }
  }

  if (!workflowID) {
    return {err: `Cannot find workflow ID for command ${command}`}
  }

  // Extract cri and k8s ver from the rest lines or use defaults.
  let ver = [];
  let cri = [];
  for (const line of lines) {
    let useParts = line.split('/e2e/use/cri/');
    if (useParts[1]) {
      cri.push(useParts[1]);
    }
    useParts = line.split('/e2e/use/k8s/');
    if (useParts[1]) {
      ver.push(useParts[1]);
    }
  }

  const inputs = {
    cri: cri.join(','),
    ver: ver.join(','),
  }

  // Add initial_ref_slug input when e2e command has two args.
  if (initialRef) {
    inputs.initial_ref_slug = initialRef.refSlug
  }

  return buildReturn('isE2E', workflowID, targetRef, inputs)
}


module.exports = {
  tryParseAbortE2eCluster,
  tryParseRunE2e
}
