import React, { useState } from "react";

import { defaultValueAction } from "./defaultValues";
import "./Actions.css";
import Input from "./Input";
import {
  Icon,
  Button,
  Card,
  H5,
  Classes,
  Intent,
  Divider,
} from "@blueprintjs/core";

function Action(props) {
  const { setSentData, actionName, action, ws } = props;
  const [formContent, setFormContent] = useState(defaultValueAction(action));
  return (
    <Card elevation={0} className="card Action">
      <>
        <div className="ActionUpperSection">
          <H5>
            <Icon
              className="HeadlineIcon"
              iconSize={17}
              icon="send-to"
              intent={Intent.PRIMARY}
            />
            {actionName}
          </H5>
          <Divider />
          <div className="InputsWrapper">
            <Input
              currentFormContent={formContent}
              setFormContent={setFormContent}
              action={action}
            />
          </div>
        </div>
        <div className="ActionLower">
          <Divider />
          <div className="ActionSendButtonWrapper">
            {/*<Button
              className="CardButton"
              intent={Intent.PRIMARY}
              rightIcon="inbox"
              disabled
              minimal
              text="View Response"
            />*/}
            <Button
              intent={Intent.PRIMARY}
              rightIcon="send-message"
              text="Send"
              className={Classes.BUTTON}
              onClick={() => {
                setSentData({
                  kind: actionName,
                  content: JSON.stringify(formContent),
                });
                ws.send(
                  JSON.stringify({
                    kind: actionName,
                    content: JSON.stringify(formContent),
                  })
                );
              }}
            />
          </div>
        </div>
      </>
    </Card>
  );
}

function Actions({ ws, setSentData, config }) {
  if (!config) {
    return null;
  }
  return (
    <>
      {Object.entries(config.actions).map(([keyName, value]) => {
        return (
          <Action
            ws={ws}
            setSentData={setSentData}
            key={keyName}
            actionName={keyName}
            action={value}
          />
        );
      })}
    </>
  );
}

export default Actions;
