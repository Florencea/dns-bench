import {
  Button,
  ButtonGroup,
  Flex,
  Form,
  Radio,
  RadioGroup,
  TextField,
  View,
} from "@adobe/react-spectrum";
import { useState } from "react";
import type { TranslationDictionary } from "../../shared/locales";
import type { ModeT } from "../../shared/types";
import { DomainInputSchema, ModeSchema } from "../../shared/types";

interface QueryFormProps {
  mode: ModeT;
  onModeChange: (mode: ModeT) => void;
  onSubmit: (host: string) => Promise<void>;
  onReset: () => void;
  isLoading: boolean;
  t: TranslationDictionary["ui"];
}

export const QueryForm = ({
  mode,
  onModeChange,
  onSubmit,
  onReset,
  isLoading,
  t,
}: QueryFormProps) => {
  const [domainValue, setDomainValue] = useState("");
  const [validationError, setValidationError] = useState<string | null>(null);

  const handleAction = async (formData: FormData) => {
    const rawHost = formData.get("host");
    const hostStr = typeof rawHost === "string" ? rawHost.trim() : "";

    const validation = DomainInputSchema.safeParse(hostStr);
    if (!validation.success) {
      setValidationError(t.domainInvalid);
      return;
    }

    setValidationError(null);
    await onSubmit(validation.data);
  };

  const handleReset = () => {
    setDomainValue("");
    setValidationError(null);
    onReset();
  };

  return (
    <View
      borderWidth="thin"
      borderColor="dark"
      borderRadius="medium"
      padding="size-250"
    >
      <Form
        isDisabled={isLoading}
        onSubmit={(e) => {
          e.preventDefault();
          const formData = new FormData(e.currentTarget);
          void handleAction(formData);
        }}
        onReset={handleReset}
      >
        <Flex alignItems="end" gap="size-200">
          <RadioGroup
            label={t.queryMode}
            name="mode"
            orientation="horizontal"
            value={mode}
            onChange={(value) => {
              const parsed = ModeSchema.safeParse(value);
              if (parsed.success) {
                onModeChange(parsed.data);
              }
            }}
          >
            <Radio value="dns">{t.dnsMode}</Radio>
            <Radio value="doh">{t.dohMode}</Radio>
          </RadioGroup>
          <TextField
            label={t.domainLabel}
            description={t.domainPlaceholder}
            name="host"
            value={domainValue}
            onChange={(val) => {
              setDomainValue(val);
              if (validationError !== null) {
                setValidationError(null);
              }
            }}
            errorMessage={validationError ?? undefined}
            validationState={validationError !== null ? "invalid" : "valid"}
          />
          <ButtonGroup>
            <Button type="submit" variant="primary" isPending={isLoading}>
              {t.submitBtn}
            </Button>
            <Button type="reset" variant="secondary">
              {t.resetBtn}
            </Button>
          </ButtonGroup>
        </Flex>
      </Form>
    </View>
  );
};
