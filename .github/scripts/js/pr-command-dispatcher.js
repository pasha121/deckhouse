const {tryParseAbortE2eCluster, tryParseRunE2e} = require("./e2e/slash_workflow_command");
const {commentCommandRecognition} = require("./comments");
const {extractCommandFromComment, reactToComment, startWorkflow} = require("./ci");

/**
 * Use pull request comment to determine a workflow to run.
 *
 * @param {object} inputs
 * @param {object} inputs.github - A pre-authenticated octokit/rest.js client with pagination plugins.
 * @param {object} inputs.context - An object containing the context of the workflow run.
 * @param {object} inputs.core - A reference to the '@actions/core' package.
 * @returns {Promise<void|*>}
 */
async function runSlashCommandForPullRequest({ github, context, core }) {
  const event = context.payload;
  const comment_id = event.comment.id;
  core.debug(`Event: ${JSON.stringify(event)}`);

  const arg = extractCommandFromComment(event.comment.body)
  const {argv} = arg
  if(arg.err) {
    return core.info(`Ignore comment: ${arg.err}.`);
  }

  let slashCommand = dispatchPullRequestCommand({arg, core, context});
  if (!slashCommand) {
    return core.info(`Ignore comment: workflow for command ${argv[0]} not found.`);
  }

  if (slashCommand.err) {
    return core.setFailed(`Cannot start workflow: ${slashCommand.err}`);
  }

  core.info(`Command detected: ${JSON.stringify(slashCommand)}`);

  const { targetRef, workflow_id } = slashCommand;
  // Git ref is malformed.
  if (!targetRef) {
    core.setFailed('targetRef is missed');
    return await reactToComment({github, context, comment_id, content: 'confused'});
  }

  // Git ref is malformed.
  if (!workflow_id) {
    core.setFailed('workflowID is missed');
    return await reactToComment({github, context, comment_id, content: 'confused'});
  }

  core.info(`Use ref '${targetRef}' for workflow.`);

  // React with rocket emoji!
  await reactToComment({github, context, comment_id, content: 'rocket'});

  // Add new issue comment and start the requested workflow.
  core.info('Add issue comment to report workflow status.');
  let response = await github.rest.issues.createComment({
    owner: context.repo.owner,
    repo: context.repo.repo,
    issue_number: event.issue.number,
    body: commentCommandRecognition(event.comment.user.login, argv[0])
  });

  if (response.status !== 201) {
    return core.setFailed(`Cannot start workflow: ${JSON.stringify(response)}`);
  }

  return await startWorkflow({github, context, core,
    workflow_id: workflow_id,
    ref: targetRef,
    inputs: {
      comment_id: '' + response.data.id,
      ...slashCommand.inputs
    },
  });
}

/**
 *
 * @param {object} arg - slash command arguments as argv [0] arg is name of command and as lines comment lines
 * @param {object} core - github core object
 * @param {object} context - github core object
 * @return {object}
 */
function dispatchPullRequestCommand({arg, core, context}){
  const { argv, lines } = arg;
  const command = argv[0];
  core.debug(`Command is ${command}`)
  core.debug(`argv is ${JSON.stringify(argv)}`)

  // TODO rewrite to some argv parse library
  const checks = [
    tryParseRunE2e,
    tryParseAbortE2eCluster
  ]

  const prNumber = context.payload.pull_request.number;
  // Construct head commit ref using pr number.
  const ref = `refs/pull/${ prNumber }/head`;
  core.notice(`Use ref: '${ref}'`)

  for (let i = 0; i < checks.length; i++) {
    const res = checks[i]({argv, lines, core, context, ref})
    if (res !== null) {
      return res;
    }
  }

  return null;
}

module.exports = {
  runSlashCommandForPullRequest,
  dispatchPullRequestCommand
}
