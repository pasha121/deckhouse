const {abortFailedE2eCommand, knownLabels, knownProviders} = require("../constants");
const {parseCommandArgumentAsRef} = require("../ci");

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
function tryParseAbortE2eCluster({argv, context, core, github}){
  const command = argv[0];
  if (command !== abortFailedE2eCommand) {
    return null;
  }

  if (argv.length !== 8) {
    let err = 'clean failed e2e cluster should have 6 arguments'
    switch (argv.length){
      case 7:
        err = 'comment id for starting e2e is required';
        break;
      case 6:
        err = 'cluster_prefix and comment id for starting e2e are required';
        break;
      case 5:
        err = 'artifact name and cluster_prefix and comment id for starting e2e are required';
        break;
      case 4:
        err = 'run id, artifact name and cluster_prefix and comment id for starting e2e are required';
        break;
      case 3:
        err = 'ran for (provider, layout, cri, k8s version), run id, artifact name and cluster_prefix and comment id for starting e2e are required';
        break;
      case 2:
        err = 'pull_request_ref, ran for (provider, layout, cri, k8s version), run id, artifact name, and cluster_prefix and comment id for starting e2e are required';
        break;
      case 1:
        err = 'ci_commit_ref_name, pull_request_ref, ran for (provider, layout, cri, k8s version), run id, artifact name, and cluster_prefix and comment id for starting e2e are required';
        break;
    }
    return {err};
  }

  const ranForSplit = argv[3].split(';').map(v => v.trim()).filter(v => !!v);
  if (ranForSplit.length !== 4) {
    let err = '"ran for" argument should have 4 parts';
    switch (ranForSplit.length) {
      case 3:
        err = 'k8s version is required';
        break;
      case 2:
        err = 'cri and k8s version are required';
        break;
      case 1:
        err = 'layout, cri and k8s version are required';
        break;
      case 0:
        err = 'provider, layout, cri and k8s version are required';
        break;
    }

    return {err};
  }

  const provider = ranForSplit[0];

  return buildReturn('isDestroyFailedE2e', `e2e-clean-${provider}.yml`,'refs/heads/main', {
      ci_commit_ref_name: argv[1],
      pull_request_ref: argv[2],
      run_id: argv[4],
      state_artifact_name: argv[5],
      cluster_prefix: argv[6],

      layout: ranForSplit[1],
      cri: ranForSplit[2],
      k8s_version: ranForSplit[3],
      issue_number: argv[7],
    },
  )
}


/**
 * Try to parse start e2e arguments
 * @param {object} inputs
 * @param {object} inputs.core - A reference to the '@actions/core' package.
 * @param {object} inputs.github - A pre-authenticated octokit/rest.js client with pagination plugins.
 * @param {object} inputs.context - A reference to context https://github.com/actions/toolkit/blob/main/packages/github/src/context.ts#L6
 * @param {string} inputs.argv - array of slash command argv[0] is commnad
 */
function tryParseRunE2e({argv, context, core, github}){
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
    targetRef = parseCommandArgumentAsRef(parts[2])
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
